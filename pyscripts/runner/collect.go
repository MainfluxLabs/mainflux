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
// output cap. Only the parent-written exit frame decides that a run is over.
// The main loop and the post-exit drain share one collector, so their limits
// cannot drift apart.
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

// accept folds one data frame in. Crossing the output cap kills the run, but
// reading continues so the stream reaches a known boundary. Frames from another
// job are dropped and not charged against this run's cap.
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

			// Only before the exit frame: after it the child has been reaped,
			// and its pid may belong to another process.
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

// drain reads a run's data stream up to the worker's flush sentinel, which
// proves the stream is frame-aligned. It reports whether the sentinel arrived
// within maxDrain; if not, the worker is retired.
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

// finish turns what was collected into the run's outcome. A cause Go knows
// about outranks whatever the child claimed, and produced is cleared to match,
// so Validate never reports a timeout as a syntax error.
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

// spoiled reports whether the worker may still hold frames from this run, in
// which case it is retired. A crossed frame does not count: the next collector
// drops it by its sequence number.
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

// reasonFor tells the run's timeout apart from the caller's cancellation, which
// arrive on the same channel, so the saved run records the real cause.
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
