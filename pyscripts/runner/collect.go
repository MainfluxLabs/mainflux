// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package runner

import (
	"context"
	"fmt"
	"time"
)

// maxDrain bounds the wait for a run's flush sentinel. Not reaching it in that
// time means a frame was left half-written and is still swallowing the stream,
// so the worker cannot safely serve another run.
const maxDrain = 500 * time.Millisecond

// collector accumulates what a run claims on the data channel and enforces the
// output cap as it reads. Everything the child writes is a claim: it holds the
// data pipe, so it can emit whatever frames it likes. Only the parent-written
// exit frame decides that a run is over.
//
// Both the main loop and the post-exit drain feed the same collector, so the
// output cap and the log limits cannot drift apart between them.
type collector struct {
	cfg     Config
	pid     int
	seq     uint64
	res     outcome
	read    int64
	over    bool
	got     bool
	overran bool
	crossed bool
	reaped  bool
	// lastFlush is the nonce of the most recent flush frame seen for this run.
	// The worker writes its flush last, after the child is dead, so in an aligned
	// stream this ends up holding the exit frame's nonce; a half-written frame
	// swallows that echo and leaves it set to something else, or unset.
	lastFlush string
	// kill ends the run this collector is reading. It is supplied rather than
	// called directly so the collector does not need to know which worker hosts
	// the run in order to signal it safely.
	kill func(int)
}

func newCollector(cfg Config, pid int, seq uint64, kill func(int)) *collector {
	return &collector{
		cfg:  cfg,
		pid:  pid,
		seq:  seq,
		kill: kill,
		res:  outcome{Result: Result{StartedAt: time.Now(), Logs: []string{}}},
	}
}

// accept folds one data frame in. Crossing the output cap kills the run and
// stops anything further being kept, but reading continues so the stream reaches
// a known boundary.
//
// A frame belonging to another job is a straggler from a run whose drain did not
// reach it. It is dropped, and deliberately not charged against this run's cap:
// charging it meant a predecessor that had written close to MaxOutput could push
// an innocent run over the limit, killing a child that had produced nothing and
// reporting an output limit the script never approached. A script cannot use
// that to evade its own cap, because its own frames carry its own seq; writing
// frames under a foreign seq only forfeits them.
func (c *collector) accept(m dataMsg) {
	if m.frame.Seq != c.seq {
		c.crossed = true
		return
	}

	// The flush sentinel is the worker's, not the script's output: it is not
	// charged against the cap and is never a log or result, only the proof the
	// stream reached a frame boundary.
	if m.frame.Kind == "flush" {
		c.lastFlush = m.frame.Nonce
		return
	}

	c.read += int64(m.size)

	if c.read > c.cfg.MaxOutput {
		if !c.over {
			c.over = true

			// Only while the child is still ours to signal. Once the exit frame
			// has arrived the worker parent has already waited on the child and
			// killed its group, and Go never waits on it, so nothing pins the
			// pid: signalling it then could land on an unrelated process group
			// that has since been given the same number - plausibly another
			// run's, in this same container.
			if !c.reaped {
				c.kill(c.pid)
			}
		}
		return
	}

	switch m.frame.Kind {
	case "log":
		if len(c.res.Logs) < c.cfg.MaxLogLines {
			c.res.Logs = append(c.res.Logs, m.frame.Text)
		}
	case "result":
		c.res.Value = m.frame.Value
		c.res.Error = m.frame.Error
		c.got = true
	}
}

// drain consumes the tail of a run's data stream up to the worker's flush
// sentinel, which carries the nonce the exit frame named. Reaching it proves the
// stream is frame-aligned at the run's end, so every frame the child wrote has
// been read and the next run starts on a boundary. Failing to within maxDrain
// means a length-prefixed frame was left half-written and is still consuming the
// stream - the worker is retired rather than handed on. It reports alignment.
func (c *collector) drain(ch <-chan dataMsg, nonce string) bool {
	// A drain only ever runs after the exit frame, by which point the child has
	// been waited on and its group killed.
	c.reaped = true

	budget := time.NewTimer(maxDrain)
	defer budget.Stop()

	for {
		if c.lastFlush == nonce {
			return true
		}

		select {
		case m := <-ch:
			c.accept(m)

		case <-budget.C:
			c.overran = true
			return false
		}
	}
}

// finish turns what was collected into the run's outcome. A cause Go knows about
// outranks whatever the child claimed, since the child may have reported success
// right before being killed.
//
// produced has to agree with that override. It says whether Error is the child's
// own verdict rather than a cause of death, and Validate relies on it to tell a
// syntax error from a run that never got to compile. Were it left as c.got, a
// check whose result arrived just before the deadline or a cancellation would
// report "timeout after 10s" as the compile verdict, and creating a perfectly
// valid script would answer 400 "script is not valid python: timeout after 10s".
func (c *collector) finish(exit ctlFrame, d *deadline) outcome {
	c.res.FinishedAt = time.Now()
	c.res.produced = c.got && !c.over && !d.fired

	switch {
	case c.over:
		c.res.Value, c.res.Error = nil, ErrOutputLimit.Error()
	case d.fired:
		c.res.Value, c.res.Error = nil, reasonFor(d.cause, c.cfg.Timeout)
	case !c.got:
		c.res.Value, c.res.Error = nil, terminationReason(exit.Signal, exit.Status, c.cfg)
	}

	return c.res
}

// spoiled reports whether the worker may still hold frames from this run. A run
// cut short, or a drain that ran out of budget with frames still arriving,
// leaves the stream in a state nobody can describe, so such a worker is retired
// rather than reused.
//
// A crossed frame is deliberately not one of those cases. It is the one kind of
// residue that *is* describable: it carries another job's sequence number, so
// the next run's collector drops it on sight. Retiring the worker for it would
// hand any caller a one-line way to empty the pool - a single forged frame per
// run, each costing a respawn that blocks on importing numpy - and would buy
// nothing the filter does not already do.
func (c *collector) spoiled(d *deadline) bool {
	return c.over || d.fired || c.overran
}

// deadline arms a run's timeout exactly once. A context's Done channel stays
// ready after it fires, so it is dropped and replaced by a bounded wait for the
// parent's exit frame; without that swap the select spins straight back into the
// timeout branch and the run is reported as an infrastructure failure.
type deadline struct {
	expire <-chan struct{}
	reap   <-chan time.Time
	timer  *time.Timer
	fired  bool
	cause  error
}

func newDeadline(ctx context.Context) *deadline {
	return &deadline{expire: ctx.Done()}
}

func (d *deadline) fire(cause error) {
	d.fired = true
	d.cause = cause
	d.expire = nil
	d.timer = time.NewTimer(reapGrace)
	d.reap = d.timer.C
}

// reasonFor names why a run was cut short. A run context carries the configured
// timeout and the caller's own cancellation on the same channel, so the two have
// to be told apart: a client that hung up mid-run would otherwise be recorded as
// having exhausted a limit it never reached, and that error is what gets
// persisted, since the run is saved on a context that outlives the request.
func reasonFor(cause error, timeout time.Duration) string {
	if cause == context.Canceled {
		return "run cancelled by the caller"
	}

	return fmt.Sprintf("timeout after %s", timeout)
}

func (d *deadline) stop() {
	if d.timer != nil {
		d.timer.Stop()
	}
}
