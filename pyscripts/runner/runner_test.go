// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	runCode     = "1 + 2"
	syntaxError = "SyntaxError: invalid syntax (line 1)"
	noPython    = "/nonexistent/python3"
)

type parent struct {
	ctl  *os.File
	data *os.File
}

type behaviour func(p parent, job jobFrame)

func answer(value any, errText string) behaviour {
	return func(p parent, job jobFrame) {
		n := fmt.Sprintf("n%d", job.Seq)
		writeFrame(p.ctl, ctlFrame{Kind: "pid", Seq: job.Seq})
		writeFrame(p.data, dataFrame{Kind: "result", Seq: job.Seq, Value: value, Error: errText})
		writeFrame(p.data, dataFrame{Kind: "flush", Seq: job.Seq, Nonce: n})
		writeFrame(p.ctl, ctlFrame{Kind: "exit", Seq: job.Seq, Nonce: n})
	}
}

func hang(children chan<- int) behaviour {
	return func(p parent, job jobFrame) {
		cmd := exec.Command("sleep", "30")
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		if err := cmd.Start(); err != nil {
			return
		}
		children <- cmd.Process.Pid

		writeFrame(p.ctl, ctlFrame{Kind: "pid", Seq: job.Seq, PID: cmd.Process.Pid})
		cmd.Wait()

		n := fmt.Sprintf("n%d", job.Seq)
		writeFrame(p.data, dataFrame{Kind: "flush", Seq: job.Seq, Nonce: n})
		writeFrame(p.ctl, ctlFrame{Kind: "exit", Seq: job.Seq, Nonce: n, Signal: int(syscall.SIGKILL)})
	}
}

func crash() behaviour {
	return func(p parent, job jobFrame) {
		p.ctl.Close()
	}
}

func newTestWorker(t *testing.T, b behaviour) *worker {
	t.Helper()

	cmd := exec.Command("sleep", "30")
	require.Nil(t, cmd.Start(), "unexpected error starting the worker process")

	pipe := func() (*os.File, *os.File) {
		r, w, err := os.Pipe()
		require.Nil(t, err, fmt.Sprintf("unexpected pipe error: %s", err))
		t.Cleanup(func() {
			r.Close()
			w.Close()
		})
		return r, w
	}

	jobR, jobW := pipe()
	ctlR, ctlW := pipe()
	dataR, dataW := pipe()
	_, apiW := pipe()

	w := &worker{
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

	if b == nil {
		jobR.Close()
	} else {
		go func() {
			for {
				var job jobFrame
				if _, err := readFrame(jobR, &job.jobHeader); err != nil {
					return
				}
				if _, err := readFrame(jobR, &job); err != nil {
					return
				}
				b(parent{ctl: ctlW, data: dataW}, job)
			}
		}()
	}

	go w.pumpCtl()
	go w.pumpData()

	t.Cleanup(w.stop)

	return w
}

func newTestPool(cfg Config, workers ...*worker) *pool {
	cfg.Python = noPython
	cfg.Workers = len(workers)

	p := &pool{cfg: cfg, logger: logger.NewMock(), free: make(chan *worker, len(workers)+1)}
	for _, w := range workers {
		p.free <- w
	}

	return p
}

func testConfig() Config {
	return Config{
		Timeout:        2 * time.Second,
		AcquireTimeout: time.Second,
		MaxOutput:      1 << 20,
		MaxLogLines:    10,
	}
}

func (p *pool) owed() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.missing
}

func TestRun(t *testing.T) {
	cases := []struct {
		desc    string
		workers func(t *testing.T) []*worker
		value   any
		free    int
		missing int
		err     error
	}{
		{
			desc: "run a script",
			workers: func(t *testing.T) []*worker {
				return []*worker{newTestWorker(t, answer(3.0, ""))}
			},
			value: json.Number("3"),
			free:  1,
			err:   nil,
		},
		{
			desc: "retry on a fresh worker when the job never reached the first",
			workers: func(t *testing.T) []*worker {
				return []*worker{newTestWorker(t, nil), newTestWorker(t, answer(3.0, ""))}
			},
			value:   json.Number("3"),
			free:    1,
			missing: 1,
			err:     nil,
		},
		{
			desc: "retry on a fresh worker when the first died while idle",
			workers: func(t *testing.T) []*worker {
				dead := newTestWorker(t, answer(1.0, ""))
				dead.cmd.Process.Kill()
				dead.cmd.Wait()
				return []*worker{dead, newTestWorker(t, answer(3.0, ""))}
			},
			value:   json.Number("3"),
			free:    1,
			missing: 1,
			err:     nil,
		},
		{
			desc: "not retry once the job reached the worker",
			workers: func(t *testing.T) []*worker {
				return []*worker{newTestWorker(t, crash()), newTestWorker(t, answer(3.0, ""))}
			},
			free:    1,
			missing: 1,
			err:     ErrWorkerFailed,
		},
		{
			desc: "fail when every attempt finds a dead worker",
			workers: func(t *testing.T) []*worker {
				return []*worker{newTestWorker(t, nil), newTestWorker(t, nil), newTestWorker(t, nil)}
			},
			free:    0,
			missing: 3,
			err:     ErrWorkerFailed,
		},
		{
			desc: "shed a request when no worker is free",
			workers: func(t *testing.T) []*worker {
				return nil
			},
			err: ErrBusy,
		},
	}

	for _, tc := range cases {
		cfg := testConfig()
		cfg.AcquireTimeout = 200 * time.Millisecond
		p := newTestPool(cfg, tc.workers(t)...)

		res, err := p.Run(context.Background(), runCode, Input{})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.err, err))
		assert.Equal(t, tc.value, res.Value, fmt.Sprintf("%s: unexpected value", tc.desc))
		assert.Equal(t, tc.free, len(p.free), fmt.Sprintf("%s: expected %d free workers got %d", tc.desc, tc.free, len(p.free)))
		assert.Eventually(t, func() bool { return p.owed() == tc.missing }, time.Second, 10*time.Millisecond,
			fmt.Sprintf("%s: expected %d workers owed got %d", tc.desc, tc.missing, p.owed()))

		p.Close()
	}
}

func TestRunTimeout(t *testing.T) {
	children := make(chan int, 1)
	cfg := testConfig()
	cfg.Timeout = 200 * time.Millisecond
	p := newTestPool(cfg, newTestWorker(t, hang(children)))
	defer p.Close()

	res, err := p.Run(context.Background(), runCode, Input{})
	require.Nil(t, err, fmt.Sprintf("unexpected run error: %s", err))
	assert.Equal(t, "timeout after 200ms", res.Error, "expected the run to time out")
	assert.Nil(t, res.Value, "expected no value from a timed out run")
	assert.Equal(t, 1, len(p.free), "expected the worker to be kept after a timeout")
	assert.False(t, res.Retired, "expected the result not to report a retired worker")

	pid := <-children
	assert.Equal(t, syscall.ESRCH, syscall.Kill(pid, 0), "expected the run to be killed")
}

func TestRunCancelled(t *testing.T) {
	children := make(chan int, 1)
	p := newTestPool(testConfig(), newTestWorker(t, hang(children)))
	defer p.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-children
		cancel()
	}()

	res, err := p.Run(ctx, runCode, Input{})
	require.Nil(t, err, fmt.Sprintf("unexpected run error: %s", err))
	assert.Equal(t, "run cancelled by the caller", res.Error, "expected the run to record the cancellation")
}

func TestValidate(t *testing.T) {
	cases := []struct {
		desc    string
		worker  func(t *testing.T) *worker
		timeout time.Duration
		verdict string
		err     error
	}{
		{
			desc:    "validate valid source",
			worker:  func(t *testing.T) *worker { return newTestWorker(t, answer(nil, "")) },
			timeout: 2 * time.Second,
			verdict: "",
			err:     nil,
		},
		{
			desc:    "validate source with a syntax error",
			worker:  func(t *testing.T) *worker { return newTestWorker(t, answer(nil, syntaxError)) },
			timeout: 2 * time.Second,
			verdict: syntaxError,
			err:     nil,
		},
		{
			desc:    "validate source whose check was killed before compiling",
			worker:  func(t *testing.T) *worker { return newTestWorker(t, hang(make(chan int, 1))) },
			timeout: 200 * time.Millisecond,
			verdict: "",
			err:     ErrWorkerFailed,
		},
	}

	for _, tc := range cases {
		cfg := testConfig()
		cfg.Timeout = tc.timeout
		p := newTestPool(cfg, tc.worker(t))

		verdict, err := p.Validate(context.Background(), runCode)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.err, err))
		assert.Equal(t, tc.verdict, verdict, fmt.Sprintf("%s: expected verdict %q got %q", tc.desc, tc.verdict, verdict))

		p.Close()
	}
}

func TestRunAfterClose(t *testing.T) {
	p := newTestPool(testConfig(), newTestWorker(t, answer(3.0, "")))
	p.Close()

	_, err := p.Run(context.Background(), runCode, Input{})
	assert.True(t, errors.Contains(err, ErrClosed), fmt.Sprintf("expected %v got %v", ErrClosed, err))
}

func TestMayRespawn(t *testing.T) {
	p := newTestPool(testConfig())
	p.cfg.Workers = 2

	assert.True(t, p.mayRespawn(), "expected the first respawn of a burst to be allowed")
	assert.True(t, p.mayRespawn(), "expected a respawn within the burst to be allowed")
	assert.False(t, p.mayRespawn(), "expected a respawn beyond the burst to be refused")

	p.mu.Lock()
	p.drawn = p.drawn.Add(-time.Duration(float64(time.Second) / respawnRate))
	p.mu.Unlock()

	assert.True(t, p.mayRespawn(), "expected the budget to refill over time")
	assert.False(t, p.mayRespawn(), "expected the refill to be bounded by the elapsed time")
}

func TestReplace(t *testing.T) {
	p := newTestPool(testConfig())
	p.cfg.Workers = 1

	p.replace()
	assert.Equal(t, 1, p.owed(), "expected a failed respawn to be owed")

	p.replace()
	assert.Equal(t, 2, p.owed(), "expected a rate-limited respawn to be owed")
}

func newSandboxPool(t *testing.T) *pool {
	t.Helper()

	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 is not installed")
	}
	if err := exec.Command(python, "-I", "-c", "import pyseccomp").Run(); err != nil {
		t.Skip("pyseccomp is not installed")
	}

	cfg := testConfig()
	cfg.Python = python
	cfg.Workers = 1
	cfg.StartTimeout = time.Minute
	cfg.BLASThreads = 1
	cfg.Limits = Limits{CPUSeconds: 5, MaxMemory: 1 << 28}

	p := &pool{cfg: cfg, logger: logger.NewMock(), free: make(chan *worker, 1)}
	w, err := p.spawn()
	require.Nil(t, err, fmt.Sprintf("unexpected error starting a sandboxed worker: %s", err))
	p.free <- w

	t.Cleanup(func() { p.Close() })

	return p
}

func TestRunSandboxed(t *testing.T) {
	cases := []struct {
		desc  string
		code  string
		value any
		err   string
	}{
		{
			desc:  "run a script",
			code:  runCode,
			value: json.Number("3"),
		},
		{
			desc: "deny opening a socket",
			code: "import socket\nsocket.socket()",
			err:  "PermissionError",
		},
		{
			desc: "deny forking a process",
			code: "import os\nos.fork()",
			err:  "PermissionError",
		},
		{
			desc: "deny writing a file",
			code: "open('/tmp/x', 'w')",
			err:  "Read-only file system",
		},
		{
			desc: "deny signalling another process through fcntl",
			code: "import os, fcntl, signal\nr, w = os.pipe()\nfcntl.fcntl(r, fcntl.F_SETOWN, os.getppid())\nfcntl.fcntl(r, 10, signal.SIGKILL)\n" +
				"fcntl.fcntl(r, fcntl.F_SETFL, fcntl.fcntl(r, fcntl.F_GETFL) | os.O_ASYNC)\nos.write(w, b'x')",
			err: "PermissionError",
		},
		{
			desc: "allow the fd handling of the standard library",
			code: "import os\nr, w = os.pipe()\nos.set_inheritable(r, True)\nos.set_blocking(r, False)\nd = os.dup(r)\n" +
				"[os.get_inheritable(d), os.get_blocking(r), os.isatty(r)]",
			value: []any{false, false, false},
		},
		{
			desc:  "allow allocating within the memory limit",
			code:  "len(bytearray(64 << 20))",
			value: json.Number("67108864"),
		},
		{
			desc: "deny allocating beyond the memory limit",
			code: "bytearray(512 << 20)",
			err:  "MemoryError",
		},
		{
			desc: "kill a script that runs past the timeout",
			code: "while True: pass",
			err:  "timeout after 2s",
		},
		{
			desc: "survive a partial frame written to the result channel",
			code: "import os\nos.write(3, b'\\x00\\x00\\x10\\x00x')",
		},
		{
			desc: "survive an oversized frame header written to the result channel",
			code: "import os\nos.write(3, b'\\xff\\xff\\xff\\xff' + b'x' * 100)",
		},
		{
			desc: "stop a script that floods its output",
			code: "while True: print('x' * 4096)",
			err:  "output limit exceeded",
		},
	}

	for _, tc := range cases {
		p := newSandboxPool(t)

		res, err := p.Run(context.Background(), tc.code, Input{})
		require.Nil(t, err, fmt.Sprintf("%s: unexpected run error: %s", tc.desc, err))
		assert.Equal(t, tc.value, res.Value, fmt.Sprintf("%s: unexpected value", tc.desc))
		assert.Contains(t, res.Error, tc.err, fmt.Sprintf("%s: unexpected error %q", tc.desc, res.Error))
		assert.False(t, res.Retired, fmt.Sprintf("%s: expected the worker to be kept", tc.desc))

		res, err = p.Run(context.Background(), runCode, Input{})
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error on the next run: %s", tc.desc, err))
		assert.Equal(t, json.Number("3"), res.Value, fmt.Sprintf("%s: expected the same worker to serve the next run", tc.desc))
	}
}

func TestRunDoesNotSeeEarlierJobs(t *testing.T) {
	p := newSandboxPool(t)

	earlier := Input{Payload: map[string]any{"pad": strings.Repeat("p", 200<<10), "secret": "leak-canary-c8e1f2"}}
	for range 3 {
		_, err := p.Run(context.Background(), runCode, earlier)
		require.Nil(t, err, fmt.Sprintf("unexpected error running an earlier job: %s", err))
	}

	scan := `
import ctypes
head, tail = b"leak-canary-", b"c8e1f2"
step, span = 1 << 16, len(head) + len(tail)
buf = bytearray(step + span)
dst = ctypes.addressof((ctypes.c_char * len(buf)).from_buffer(buf))
found = False
for line in open("/proc/self/maps"):
    f = line.split()
    if not f[1].startswith("rw"):
        continue
    lo, hi = (int(x, 16) for x in f[0].split("-"))
    at = lo
    while at < hi and not found:
        n = min(step + span, hi - at)
        ctypes.memmove(dst, at, n)
        i = buf.find(head, 0, n)
        while i >= 0 and not found:
            found = i + span <= n and buf.startswith(tail, i + len(head))
            i = buf.find(head, i + 1, n)
        at += step
found
`
	res, err := p.Run(context.Background(), scan, Input{})
	require.Nil(t, err, fmt.Sprintf("unexpected error running the second job: %s", err))
	assert.Empty(t, res.Error, "expected the memory scan to complete")
	assert.Equal(t, false, res.Value, "expected a run not to find an earlier job's payload in its memory")
}
