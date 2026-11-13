// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pyscripts"
)

const (
	sandboxSeccomp = "seccomp"

	// reapGrace bounds how long the pool waits for a worker to report a killed
	// run as finished before treating the worker as broken.
	reapGrace = 5 * time.Second

	// maxWorkerAttempts bounds how many workers one job will try before giving
	// up, so a pool full of corpses fails fast instead of looping.
	maxWorkerAttempts = 3

	// maxStderrLog bounds how much of one worker's stderr is copied into the
	// service log, so a worker failing in a loop cannot fill it.
	maxStderrLog = 64 << 10

	// backfillInterval is how often the pool retries workers it failed to
	// start. It also bounds how long a rate-limited respawn leaves the pool
	// short.
	backfillInterval = 5 * time.Second

	// respawnRate caps sustained worker respawns per second, over a burst of
	// one full pool. A script can force a respawn, and each one imports numpy.
	respawnRate = 1.0

	// orphanGrace bounds how long a cancelled job waits for the pid of a child
	// the worker may already have forked for it, so that child can be killed
	// rather than left running for nobody.
	orphanGrace = 500 * time.Millisecond
)

// hideProcess and reapOrphans are variables so hardening.go can install the
// Linux implementations. Elsewhere they are no-ops.
var (
	hideProcess = func() error { return nil }
	reapOrphans = func(logger.Logger) {}

	// parentOf reports a process's parent, or -1 where that cannot be known.
	// Unknown means "assume it is still ours", so the fallback in killRun keeps
	// working off Linux.
	parentOf = func(int) int { return -1 }
)

// These name the same failures as the pyscripts errors they wrap, so callers can
// match on either: code in this package compares against these, and the service
// and HTTP layers against the exported ones.
var (
	ErrWorkerFailed = pyscripts.ErrRunnerFailed
	ErrBusy         = pyscripts.ErrRunnerBusy
	ErrOutputLimit  = pyscripts.ErrOutputLimit
	ErrClosed       = pyscripts.ErrRunnerClosed
)

type Limits struct {
	CPUSeconds  int   `json:"cpu_seconds"`
	MaxMemory   int64 `json:"max_memory"`
	MaxFileSize int64 `json:"max_file_size"`
}

type Config struct {
	Python  string
	Workers int
	// Timeout bounds a single run, measured from the moment a worker is obtained.
	Timeout time.Duration
	// AcquireTimeout bounds how long a caller waits for a free worker before the
	// pool sheds the request. Without it a saturated pool queues indefinitely.
	AcquireTimeout time.Duration
	// StartTimeout bounds the handshake with a newly spawned worker, so one
	// that never reports ready cannot hang startup.
	StartTimeout time.Duration
	MaxOutput    int64
	MaxLogLines  int
	BLASThreads  int
	Limits       Limits
}

// Input and Result are aliased from pyscripts, which owns the contract.
type (
	Input  = pyscripts.Input
	Result = pyscripts.Result
)

// outcome is a Result plus produced, which records whether the child reported
// an outcome rather than being killed first. Validate uses it to tell a syntax
// error from a check that never compiled.
type outcome struct {
	Result
	produced bool
}

type pool struct {
	cfg    Config
	logger logger.Logger
	free   chan *worker
	mu     sync.Mutex
	closed bool
	// missing counts workers owed because a spawn failed. Such failures are
	// often transient, so the deficit is retried.
	missing int

	// tokens and drawn implement the respawn budget as a leaky bucket, refilled
	// at respawnRate and capped at one full pool so an ordinary burst - every
	// worker dying at once - is still replaced immediately.
	tokens float64

	drawn time.Time
}

func New(cfg Config, log logger.Logger) (pyscripts.Runner, error) {
	if cfg.Workers < 1 {
		return nil, errors.Wrap(ErrWorkerFailed, errors.New("at least one worker is required"))
	}

	if err := hideProcess(); err != nil {
		log.Warn(fmt.Sprintf("could not hide this process from /proc: %s - scripts may be able to read the service environment", err))
	}

	reapOrphans(log)

	p := &pool{cfg: cfg, logger: log, free: make(chan *worker, cfg.Workers)}

	// A pool short of workers still serves while backfill makes up the
	// shortfall. A pool with none cannot.
	for i := 0; i < cfg.Workers; i++ {
		w, err := p.spawn()
		if err != nil {
			p.missing++
			log.Warn(fmt.Sprintf("starting a python worker failed: %s", err))
			continue
		}
		p.free <- w
	}

	if p.missing == cfg.Workers {
		p.Close()
		return nil, errors.Wrap(ErrWorkerFailed, errors.New("no python worker could be started"))
	}

	if p.missing > 0 {
		log.Warn(fmt.Sprintf("started with %d of %d python workers; the rest will be retried every %s",
			cfg.Workers-p.missing, cfg.Workers, backfillInterval))
	}

	go p.backfill()

	return p, nil
}

// backfill tops the pool back up to its configured size.
func (p *pool) backfill() {
	t := time.NewTicker(backfillInterval)
	defer t.Stop()

	for range t.C {
		// Clear the whole deficit each tick, one spawn at a time.
		for {
			p.mu.Lock()
			owed, closed := p.missing, p.closed
			p.mu.Unlock()

			if closed {
				return
			}

			if owed == 0 {
				break
			}

			w, err := p.spawn()
			if err != nil {
				// Leave the rest owed until the next tick: a spawn that just
				// failed is likely to fail again.
				p.logger.Warn(fmt.Sprintf("replacing a missing python worker failed: %s", err))
				break
			}

			p.mu.Lock()
			if p.closed {
				p.mu.Unlock()
				w.stop()
				return
			}
			p.missing--
			short := p.missing
			p.mu.Unlock()

			p.release(w)
			p.logger.Info(fmt.Sprintf("replaced a missing python worker; %d still short", short))
		}
	}
}

func (p *pool) Run(ctx context.Context, code string, in Input) (Result, error) {
	res, err := p.onWorker(ctx, func(w *worker, runCtx context.Context) (outcome, error) {
		return w.run(runCtx, code, in, p.cfg)
	})

	return res.Result, err
}

func (p *pool) Validate(ctx context.Context, code string) (string, error) {
	res, err := p.onWorker(ctx, func(w *worker, runCtx context.Context) (outcome, error) {
		return w.check(runCtx, code, p.cfg)
	})
	if err != nil {
		return "", err
	}

	// A check that never reported was killed before compiling, so Error holds
	// its cause of death, not a verdict on the source.
	if !res.produced {
		return "", errors.Wrap(ErrWorkerFailed, errors.New(res.Error))
	}

	return res.Error, nil
}

// onWorker performs one job on a pooled worker, retrying on a fresh one when
// the worker it drew was already dead. Only ErrWorkerFailed is retried: a
// script that raised, timed out or hit a limit has genuinely run.
func (p *pool) onWorker(ctx context.Context, job func(*worker, context.Context) (outcome, error)) (outcome, error) {
	var err error

	// One budget bounds the whole request, not each attempt. The real ceiling
	// is the budget plus reapGrace, since awaitFinish waits that long after
	// killing the child.
	budget, stopBudget := context.WithTimeout(ctx, p.cfg.AcquireTimeout+p.cfg.Timeout)
	defer stopBudget()

	for attempt := 0; attempt < maxWorkerAttempts; attempt++ {
		var w *worker
		if w, err = p.acquire(budget, ctx); err != nil {
			return outcome{}, err
		}

		if !w.healthy() {
			p.logger.Warn("discarding a python worker that died while idle in the pool")
			p.discard(w)
			continue
		}

		runCtx, cancel := context.WithTimeout(budget, p.cfg.Timeout)
		res, jobErr := job(w, runCtx)
		cancel()

		if jobErr == nil {
			if w.crossed {
				p.logger.Warn("dropped a python data frame stamped with another job's sequence number")
			}

			if n := w.garbled.Swap(0); n > 0 {
				p.logger.Warn(fmt.Sprintf("dropped %d undecodable python data frame(s): a script wrote to its result channel by hand", n))
			}

			if w.dirty {
				p.discard(w)
			} else {
				p.release(w)
			}

			return res, nil
		}

		err = jobErr
		ran := w.dispatched
		p.discard(w)

		// Once the job is dispatched, a retry would run the script again, side
		// effects included.
		if ctx.Err() != nil || budget.Err() != nil || ran || !errors.Contains(err, ErrWorkerFailed) {
			return outcome{}, err
		}
	}

	if err == nil {
		err = errors.New("no healthy python worker available")
	}

	return outcome{}, errors.Wrap(ErrWorkerFailed, err)
}

func (p *pool) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()

	close(p.free)
	for w := range p.free {
		w.stop()
	}
	return nil
}

// acquire waits for a free worker. It takes the request's remaining budget to
// wait against and the caller's own context to interpret the outcome: a lapsed
// budget means the pool is saturated and the request should be shed as ErrBusy,
// whereas a caller that has gone away is reported as its own cancellation.
func (p *pool) acquire(budget, caller context.Context) (*worker, error) {
	wait, cancel := context.WithTimeout(budget, p.cfg.AcquireTimeout)
	defer cancel()

	select {
	case w, ok := <-p.free:
		if !ok {
			return nil, ErrClosed
		}
		return w, nil
	case <-wait.Done():
		if caller.Err() != nil {
			return nil, caller.Err()
		}
		return nil, ErrBusy
	}
}

func (p *pool) release(w *worker) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		w.stop()
		return
	}

	p.free <- w
}

func (p *pool) discard(w *worker) {
	w.stop()

	p.mu.Lock()
	closed := p.closed
	p.mu.Unlock()

	if closed {
		return
	}

	// Replaced off the request path, so the caller does not wait for a new
	// worker to import numpy.
	go p.replace()
}

// replace brings the pool back to size after a worker is retired, recording a
// deficit for backfill to retry if it cannot.
func (p *pool) replace() {
	if !p.mayRespawn() {
		p.mu.Lock()
		p.missing++
		p.mu.Unlock()

		p.logger.Warn(fmt.Sprintf("python workers are being retired faster than %g/s; leaving the shortfall to backfill", respawnRate))
		return
	}

	repl, err := p.spawn()
	if err != nil {
		p.mu.Lock()
		p.missing++
		p.mu.Unlock()

		p.logger.Warn(fmt.Sprintf("respawning python worker failed: %s - will retry in %s", err, backfillInterval))
		return
	}

	p.release(repl)
}

// mayRespawn draws from the respawn budget, refilling it for the time elapsed
// since the last draw.
func (p *pool) mayRespawn() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	burst := float64(p.cfg.Workers)

	if p.drawn.IsZero() {
		p.tokens = burst
	} else {
		p.tokens += now.Sub(p.drawn).Seconds() * respawnRate
	}

	if p.tokens > burst {
		p.tokens = burst
	}
	p.drawn = now

	if p.tokens < 1 {
		return false
	}

	p.tokens--
	return true
}

// spawn starts one worker. Every pipe it opens is closed if any step fails,
// since a leak would recur on every worker death.
func (p *pool) spawn() (w *worker, err error) {
	var opened []*os.File

	defer func() {
		if err == nil {
			return
		}
		for _, f := range opened {
			f.Close()
		}
	}()

	pipe := func() (*os.File, *os.File, error) {
		r, wr, e := os.Pipe()
		if e != nil {
			return nil, nil, e
		}
		opened = append(opened, r, wr)
		return r, wr, nil
	}

	jobR, jobW, err := pipe()
	if err != nil {
		return nil, err
	}

	ctlR, ctlW, err := pipe()
	if err != nil {
		return nil, err
	}

	dataR, dataW, err := pipe()
	if err != nil {
		return nil, err
	}

	apiR, apiW, err := pipe()
	if err != nil {
		return nil, err
	}

	errR, errW, err := pipe()
	if err != nil {
		return nil, err
	}

	// -I isolates the interpreter and implies -E, so bytecode writing is
	// disabled with -B rather than PYTHONDONTWRITEBYTECODE.
	cmd := exec.Command(p.cfg.Python, "-I", "-B", "-c", workerSrc)
	cmd.Stdin = jobR
	cmd.Stdout = ctlW
	cmd.Stderr = errW
	cmd.ExtraFiles = []*os.File{dataW, apiR}
	cmd.Env = workerEnv(p.cfg)

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	jobR.Close()
	ctlW.Close()
	dataW.Close()
	apiR.Close()
	errW.Close()

	w = &worker{
		cmd:    cmd,
		jobs:   jobW,
		ctl:    ctlR,
		data:   dataR,
		api:    apiW,
		ctlCh:  make(chan ctlFrame, 8),
		dataCh: make(chan dataMsg, 256),
		errCh:  make(chan error, 2),
		done:   make(chan struct{}),
	}

	go w.drainStderr(errR, p.logger)
	go w.pumpCtl()
	go w.pumpData()

	var ready ctlFrame
	select {
	case ready = <-w.ctlCh:
	case err := <-w.errCh:
		w.stop()
		return nil, err
	case <-time.After(p.cfg.StartTimeout):
		w.stop()
		return nil, errors.Wrap(ErrWorkerFailed, errors.New("worker did not report ready"))
	}

	p.logger.Info(fmt.Sprintf("python worker ready: pid=%d %s", cmd.Process.Pid, ready.Loaded))

	// A worker without seccomp or /proc hiding would run untrusted code
	// unconfined, so it is refused and the service fails to boot.
	if ready.Sandbox != sandboxSeccomp {
		w.stop()
		return nil, errors.Wrap(ErrWorkerFailed, errors.New(fmt.Sprintf("worker reported no seccomp filter (%s)", ready.Sandbox)))
	}

	if !ready.Hidden {
		w.stop()
		return nil, errors.Wrap(ErrWorkerFailed, errors.New("worker could not hide its /proc entry"))
	}

	return w, nil
}

func workerEnv(cfg Config) []string {
	n := strconv.Itoa(cfg.BLASThreads)
	return []string{
		"OMP_NUM_THREADS=" + n,
		"OPENBLAS_NUM_THREADS=" + n,
		"MKL_NUM_THREADS=" + n,
		"NUMEXPR_NUM_THREADS=" + n,
	}
}

type worker struct {
	cmd   *exec.Cmd
	jobs  *os.File
	ctl   *os.File
	data  *os.File
	api   *os.File
	seq   uint64
	dirty bool
	// crossed records that the last run saw a frame stamped with another job's
	// sequence number. The frame is dropped, but logged.
	crossed bool
	// dispatched records that the job frame reached the worker. A failure after
	// that point must not be retried on another worker: the script may already
	// have run, and retrying runs it again.
	dispatched bool

	// garbled counts frames this worker's children wrote that could not be
	// decoded. They are dropped rather than fatal, so the count is the only
	// evidence they happened.
	garbled atomic.Int64

	// child is the pid of the run this worker is hosting, or 0. The run has its
	// own session, so stopping the worker does not reach it. Read by stop,
	// which can run on another goroutine.
	child atomic.Int64

	// done is closed when the worker is stopped, so its pump goroutines can give
	// up on a channel nobody will read again.
	done     chan struct{}
	stopOnce sync.Once

	ctlCh  chan ctlFrame
	dataCh chan dataMsg
	errCh  chan error
}

func (w *worker) check(ctx context.Context, code string, cfg Config) (outcome, error) {
	return w.exchange(ctx, jobFrame{Code: code, Check: true, Limits: cfg.Limits}, cfg)
}

func (w *worker) run(ctx context.Context, code string, in Input, cfg Config) (outcome, error) {
	return w.exchange(ctx, jobFrame{Input: in, Limits: cfg.Limits, Code: code}, cfg)
}

// exchange runs one job to completion: hand it over, wait for the worker to
// acknowledge it, then gather the outcome.
func (w *worker) exchange(ctx context.Context, job jobFrame, cfg Config) (outcome, error) {
	w.seq++
	job.Seq = w.seq

	started := time.Now()
	w.dispatched = false
	w.crossed = false
	w.flushData()

	// A job frame can exceed the pipe buffer, so the handoff itself blocks if the
	// worker is alive but not reading. Pipes honour deadlines, which is the only
	// way to make this cancellable.
	if dl, ok := ctx.Deadline(); ok {
		w.jobs.SetWriteDeadline(dl)
		defer w.jobs.SetWriteDeadline(time.Time{})
	}

	if err := writeFrame(w.jobs, job); err != nil {
		// A handoff that failed because the caller's context ended is not a
		// broken worker. Reporting it as one would retry the job on a fresh
		// worker for a request that is already over, running it again.
		if cause := ctx.Err(); cause != nil {
			w.dirty = true
			return abandoned(cfg, started, cause), nil
		}

		return outcome{}, errors.Wrap(ErrWorkerFailed, err)
	}

	// Past this point the worker has the job, so onWorker must not hand it to
	// another worker. A write that failed above is different: the frame never
	// arrived, so nothing ran and a retry is exactly right.
	w.dispatched = true

	pid, err := w.awaitStart(ctx, job.Seq)
	if err != nil {
		if cause := ctx.Err(); cause != nil {
			w.dirty = true
			return abandoned(cfg, started, cause), nil
		}

		return outcome{}, err
	}

	// Recorded for the whole run so stop can reach it, and cleared on the way
	// out so a later teardown cannot signal a pid that has since been recycled.
	w.child.Store(int64(pid))
	defer w.child.Store(0)

	return w.awaitFinish(ctx, job.Seq, pid, cfg)
}

// flushData discards frames a cut-short run left on the data channel, so the
// next run never sees them.
func (w *worker) flushData() {
	for {
		select {
		case <-w.dataCh:
		default:
			return
		}
	}
}

// abandoned records a run the caller stopped waiting for after the job reached
// the worker, so the run is still saved and answered like the equivalent
// timeout.
func abandoned(cfg Config, started time.Time, cause error) outcome {
	return outcome{Result: Result{
		Logs:       []string{},
		Error:      reasonFor(cause, cfg.Timeout),
		StartedAt:  started,
		FinishedAt: time.Now(),
	}}
}

// awaitStart waits for the worker to acknowledge the job and report the pid it
// forked. A job answered without forking reports pid 0, so the handshake is
// complete on receiving the frame rather than on a non-zero value.
func (w *worker) awaitStart(ctx context.Context, seq uint64) (int, error) {
	for {
		select {
		case f := <-w.ctlCh:
			if f.Seq != seq {
				continue
			}

			if f.Kind == "pid" {
				return f.PID, nil
			}
		case err := <-w.errCh:
			w.killPending(seq)
			return 0, errors.Wrap(ErrWorkerFailed, err)
		case <-ctx.Done():
			w.killPending(seq)
			return 0, ctx.Err()
		}
	}
}

// killPending kills a child the worker may already have forked for a job this
// caller stopped waiting for. The child has its own session, so retiring the
// worker does not reach it. The wait is bounded, and draining errCh is safe
// because this worker is always discarded afterwards.
func (w *worker) killPending(seq uint64) {
	t := time.NewTimer(orphanGrace)
	defer t.Stop()

	for {
		select {
		case f := <-w.ctlCh:
			if f.Kind != "pid" || f.Seq != seq {
				continue
			}

			if f.PID > 0 {
				w.killRun(f.PID)
			}

			return

		case <-w.errCh:
			return

		case <-t.C:
			return
		}
	}
}

// awaitFinish gathers data frames until the parent reports the run over, killing
// the run if the deadline passes first.
func (w *worker) awaitFinish(ctx context.Context, seq uint64, pid int, cfg Config) (outcome, error) {
	c := newCollector(cfg, pid, seq, w.killRun)

	d := newDeadline(ctx)
	defer d.stop()

	for {
		select {
		case m := <-w.dataCh:
			c.accept(m)

		case f := <-w.ctlCh:
			if f.Kind != "exit" || f.Seq != seq {
				continue
			}

			aligned := c.drain(w.dataCh, f.Nonce)
			w.dirty = c.spoiled(d) || !aligned
			w.crossed = c.crossed

			return c.finish(f, d), nil

		case <-w.errCh:
			// The child can break fd 3 itself. The run happened, so it is
			// recorded as failed and the worker retired.
			w.killRun(pid)
			w.dirty = true
			c.res.FinishedAt = time.Now()
			c.res.Value = nil
			c.res.Error = "run did not complete: the worker stopped responding"
			return c.res, nil

		case <-d.expire:
			d.fire(ctx.Err())
			w.killRun(pid)

		case <-d.reap:
			w.killRun(pid)
			return outcome{}, errors.Wrap(ErrWorkerFailed, errors.New("worker did not report the killed run as finished"))
		}
	}
}

// healthy reports whether this worker can still take a job, catching one that
// died while idle.
func (w *worker) healthy() bool {
	select {
	case <-w.errCh:
		return false
	default:
	}

	if w.cmd.Process == nil {
		return false
	}

	return w.cmd.Process.Signal(syscall.Signal(0)) == nil
}

// killRun kills one run's process group. Before the child reaches setsid it is
// still in the worker's group, so its bare pid is signalled instead, but only
// while the pid is still parented by this worker or parentage is unknown, since
// a reaped pid can be reused.
func (w *worker) killRun(pid int) {
	if pid <= 0 {
		return
	}

	if syscall.Kill(-pid, syscall.SIGKILL) == nil {
		return
	}

	if owner := parentOf(pid); owner < 0 || (w.cmd.Process != nil && owner == w.cmd.Process.Pid) {
		syscall.Kill(pid, syscall.SIGKILL)
	}
}

func (w *worker) stop() {
	w.stopOnce.Do(func() { close(w.done) })

	// Before the pipes go, so a run that outlived its request dies with the
	// worker rather than being left in its own session with nothing holding it.
	w.killRun(int(w.child.Swap(0)))

	w.jobs.Close()
	w.ctl.Close()
	w.data.Close()
	w.api.Close()

	// Released rather than waited on: reapOrphans is the only reaper, so waiting
	// here would race it for the exit status.
	if w.cmd.Process != nil {
		w.cmd.Process.Kill()
		w.cmd.Process.Release()
	}
}

// drainStderr copies a worker's own stderr into the service log, capped at
// maxStderrLog. Runs have /dev/null on fd 2, so nothing they write arrives
// here.
func (w *worker) drainStderr(r *os.File, log logger.Logger) {
	defer r.Close()

	var logged int64
	buf := make([]byte, 4096)

	for {
		n, err := r.Read(buf)
		if n > 0 && logged <= maxStderrLog {
			logged += int64(n)
			log.Error(fmt.Sprintf("python worker stderr: %s", string(buf[:n])))

			if logged > maxStderrLog {
				log.Error("python worker stderr: further output suppressed")
			}
		}
		if err != nil {
			return
		}
	}
}
