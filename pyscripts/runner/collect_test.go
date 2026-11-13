// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package runner

import (
	"context"
	"fmt"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	testSeq = uint64(7)
	testPID = 4242
	nonce   = "nonce"
)

func collectConfig() Config {
	return Config{
		Timeout:     10 * time.Second,
		MaxOutput:   100,
		MaxLogLines: 2,
		Limits:      Limits{CPUSeconds: 5},
	}
}

func frameMsg(f dataFrame, size int) dataMsg {
	return dataMsg{frame: f, size: size}
}

func TestCollectorAccept(t *testing.T) {
	cases := []struct {
		desc      string
		msgs      []dataMsg
		reaped    bool
		logs      []string
		value     any
		got       bool
		crossed   bool
		over      bool
		read      int64
		lastFlush string
		kills     []int
	}{
		{
			desc: "keep logs up to the line limit",
			msgs: []dataMsg{
				frameMsg(dataFrame{Kind: "log", Seq: testSeq, Text: "a"}, 10),
				frameMsg(dataFrame{Kind: "log", Seq: testSeq, Text: "b"}, 10),
				frameMsg(dataFrame{Kind: "log", Seq: testSeq, Text: "c"}, 10),
			},
			logs: []string{"a", "b"},
			read: 30,
		},
		{
			desc:  "record the result",
			msgs:  []dataMsg{frameMsg(dataFrame{Kind: "result", Seq: testSeq, Value: 3.0}, 10)},
			logs:  []string{},
			value: 3.0,
			got:   true,
			read:  10,
		},
		{
			desc:    "drop a frame from another job without charging it",
			msgs:    []dataMsg{frameMsg(dataFrame{Kind: "log", Seq: testSeq + 1, Text: "a"}, 90)},
			logs:    []string{},
			crossed: true,
		},
		{
			desc:      "record a flush without charging it",
			msgs:      []dataMsg{frameMsg(dataFrame{Kind: "flush", Seq: testSeq, Nonce: nonce}, 90)},
			logs:      []string{},
			lastFlush: nonce,
		},
		{
			desc: "kill the run once when output exceeds the cap",
			msgs: []dataMsg{
				frameMsg(dataFrame{Kind: "log", Seq: testSeq, Text: "a"}, 60),
				frameMsg(dataFrame{Kind: "log", Seq: testSeq, Text: "b"}, 60),
				frameMsg(dataFrame{Kind: "log", Seq: testSeq, Text: "c"}, 60),
			},
			logs:  []string{"a"},
			over:  true,
			read:  180,
			kills: []int{testPID},
		},
		{
			desc: "not kill a reaped run that exceeds the cap",
			msgs: []dataMsg{
				frameMsg(dataFrame{Kind: "log", Seq: testSeq, Text: "a"}, 60),
				frameMsg(dataFrame{Kind: "log", Seq: testSeq, Text: "b"}, 60),
			},
			reaped: true,
			logs:   []string{"a"},
			over:   true,
			read:   120,
		},
	}

	for _, tc := range cases {
		var kills []int
		c := newCollector(collectConfig(), testPID, testSeq, func(pid int) { kills = append(kills, pid) })
		c.reaped = tc.reaped

		for _, m := range tc.msgs {
			c.accept(m)
		}

		assert.Equal(t, tc.logs, c.res.Logs, fmt.Sprintf("%s: unexpected logs", tc.desc))
		assert.Equal(t, tc.value, c.res.Value, fmt.Sprintf("%s: unexpected value", tc.desc))
		assert.Equal(t, tc.got, c.got, fmt.Sprintf("%s: unexpected got", tc.desc))
		assert.Equal(t, tc.crossed, c.crossed, fmt.Sprintf("%s: unexpected crossed", tc.desc))
		assert.Equal(t, tc.over, c.over, fmt.Sprintf("%s: unexpected over", tc.desc))
		assert.Equal(t, tc.read, c.read, fmt.Sprintf("%s: unexpected bytes read", tc.desc))
		assert.Equal(t, tc.lastFlush, c.lastFlush, fmt.Sprintf("%s: unexpected flush nonce", tc.desc))
		assert.Equal(t, tc.kills, kills, fmt.Sprintf("%s: unexpected kills", tc.desc))
	}
}

func TestCollectorDrain(t *testing.T) {
	cases := []struct {
		desc    string
		msgs    []dataMsg
		aligned bool
		logs    []string
	}{
		{
			desc: "drain up to the flush sentinel",
			msgs: []dataMsg{
				frameMsg(dataFrame{Kind: "log", Seq: testSeq, Text: "a"}, 10),
				frameMsg(dataFrame{Kind: "flush", Seq: testSeq, Nonce: nonce}, 10),
				frameMsg(dataFrame{Kind: "log", Seq: testSeq, Text: "b"}, 10),
			},
			aligned: true,
			logs:    []string{"a"},
		},
		{
			desc:    "give up on a flush with another nonce",
			msgs:    []dataMsg{frameMsg(dataFrame{Kind: "flush", Seq: testSeq, Nonce: "other"}, 10)},
			aligned: false,
			logs:    []string{},
		},
		{
			desc:    "give up on a flush from another job",
			msgs:    []dataMsg{frameMsg(dataFrame{Kind: "flush", Seq: testSeq + 1, Nonce: nonce}, 10)},
			aligned: false,
			logs:    []string{},
		},
	}

	for _, tc := range cases {
		ch := make(chan dataMsg, len(tc.msgs))
		for _, m := range tc.msgs {
			ch <- m
		}

		c := newCollector(collectConfig(), testPID, testSeq, func(int) {})
		aligned := c.drain(ch, nonce)

		assert.Equal(t, tc.aligned, aligned, fmt.Sprintf("%s: expected aligned %t got %t", tc.desc, tc.aligned, aligned))
		assert.Equal(t, !tc.aligned, c.overran, fmt.Sprintf("%s: unexpected overran", tc.desc))
		assert.True(t, c.reaped, fmt.Sprintf("%s: expected the run to be marked reaped", tc.desc))
		assert.Equal(t, tc.logs, c.res.Logs, fmt.Sprintf("%s: unexpected logs", tc.desc))
	}
}

func TestCollectorFinish(t *testing.T) {
	result := func(value any, errText string) dataMsg {
		return frameMsg(dataFrame{Kind: "result", Seq: testSeq, Value: value, Error: errText}, 10)
	}

	cases := []struct {
		desc     string
		msgs     []dataMsg
		cause    error
		exit     ctlFrame
		value    any
		err      string
		produced bool
	}{
		{
			desc:     "finish a successful run",
			msgs:     []dataMsg{result(3.0, "")},
			value:    3.0,
			produced: true,
		},
		{
			desc:     "finish a run whose script raised",
			msgs:     []dataMsg{result(nil, "ValueError: x")},
			err:      "ValueError: x",
			produced: true,
		},
		{
			desc: "finish a run over the output cap",
			msgs: []dataMsg{result(3.0, ""), frameMsg(dataFrame{Kind: "log", Seq: testSeq}, 200)},
			err:  ErrOutputLimit.Error(),
		},
		{
			desc:  "finish a run that timed out after reporting",
			msgs:  []dataMsg{result(3.0, "")},
			cause: context.DeadlineExceeded,
			err:   "timeout after 10s",
		},
		{
			desc:  "finish a run cancelled by the caller",
			cause: context.Canceled,
			err:   "run cancelled by the caller",
		},
		{
			desc: "finish a run killed without a result",
			exit: ctlFrame{Kind: "exit", Seq: testSeq, Signal: int(syscall.SIGXCPU)},
			err:  "cpu time limit exceeded (5s)",
		},
	}

	for _, tc := range cases {
		c := newCollector(collectConfig(), testPID, testSeq, func(int) {})
		for _, m := range tc.msgs {
			c.accept(m)
		}

		d := newDeadline(context.Background())
		if tc.cause != nil {
			d.fire(tc.cause)
		}

		res := c.finish(tc.exit, d)
		d.stop()

		assert.Equal(t, tc.value, res.Value, fmt.Sprintf("%s: unexpected value", tc.desc))
		assert.Equal(t, tc.err, res.Error, fmt.Sprintf("%s: unexpected error", tc.desc))
		assert.Equal(t, tc.produced, res.produced, fmt.Sprintf("%s: unexpected produced", tc.desc))
		assert.False(t, res.FinishedAt.IsZero(), fmt.Sprintf("%s: expected a finish time", tc.desc))
	}
}

func TestCollectorSpoiled(t *testing.T) {
	cases := []struct {
		desc    string
		over    bool
		fired   bool
		overran bool
		crossed bool
		spoiled bool
	}{
		{desc: "keep a worker after a clean run", spoiled: false},
		{desc: "keep a worker after a frame from another job", crossed: true, spoiled: false},
		{desc: "retire a worker after the output cap", over: true, spoiled: true},
		{desc: "retire a worker after a timeout", fired: true, spoiled: true},
		{desc: "retire a worker whose drain overran", overran: true, spoiled: true},
	}

	for _, tc := range cases {
		c := &collector{over: tc.over, overran: tc.overran, crossed: tc.crossed}
		d := &deadline{fired: tc.fired}

		spoiled := c.spoiled(d)
		assert.Equal(t, tc.spoiled, spoiled, fmt.Sprintf("%s: expected spoiled %t got %t", tc.desc, tc.spoiled, spoiled))
	}
}

func TestDeadlineFire(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	d := newDeadline(ctx)
	assert.NotNil(t, d.expire, "expected an armed deadline")
	assert.Nil(t, d.reap, "expected no reap wait before firing")

	d.fire(ctx.Err())
	defer d.stop()

	assert.True(t, d.fired, "expected the deadline to be fired")
	assert.Equal(t, context.Canceled, d.cause, "expected the cause to be kept")
	assert.Nil(t, d.expire, "expected the expired channel to be dropped")
	assert.NotNil(t, d.reap, "expected a bounded reap wait")
}
