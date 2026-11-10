// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package runner

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rawFrame(payload []byte) []byte {
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(payload)))
	return append(hdr[:], payload...)
}

func TestWriteFrame(t *testing.T) {
	var buf bytes.Buffer
	sent := ctlFrame{Kind: "exit", Seq: 3, Status: 1, Nonce: "n3"}

	err := writeFrame(&buf, sent)
	require.Nil(t, err, fmt.Sprintf("unexpected write error: %s", err))

	var got ctlFrame
	n, err := readFrame(&buf, &got)
	require.Nil(t, err, fmt.Sprintf("unexpected read error: %s", err))

	payload, _ := json.Marshal(sent)
	assert.Equal(t, len(payload), n, "expected the payload size to be reported")
	assert.Equal(t, sent, got, "expected the frame to round-trip")
}

func TestReadFrame(t *testing.T) {
	valid, err := json.Marshal(dataFrame{Kind: "log", Seq: 1, Text: "hello"})
	require.Nil(t, err, fmt.Sprintf("unexpected marshal error: %s", err))

	var oversized [4]byte
	binary.BigEndian.PutUint32(oversized[:], maxFrame+1)

	cases := []struct {
		desc   string
		stream []byte
		size   int
		err    error
	}{
		{
			desc:   "read a valid frame",
			stream: rawFrame(valid),
			size:   len(valid),
			err:    nil,
		},
		{
			desc:   "read a frame with an undecodable payload",
			stream: rawFrame([]byte("not json")),
			size:   len("not json"),
			err:    errBadFrame,
		},
		{
			desc:   "read a frame longer than the ceiling",
			stream: oversized[:],
			size:   0,
			err:    ErrWorkerFailed,
		},
		{
			desc:   "read a truncated frame",
			stream: rawFrame(valid)[:6],
			size:   0,
			err:    io.ErrUnexpectedEOF,
		},
		{
			desc:   "read an empty stream",
			stream: nil,
			size:   0,
			err:    io.EOF,
		},
	}

	for _, tc := range cases {
		var f dataFrame
		n, err := readFrame(bytes.NewReader(tc.stream), &f)
		assert.True(t, errors.Is(err, tc.err), fmt.Sprintf("%s: expected %v got %v", tc.desc, tc.err, err))
		assert.Equal(t, tc.size, n, fmt.Sprintf("%s: expected size %d got %d", tc.desc, tc.size, n))
	}
}

func TestReadFrameAfterBadFrame(t *testing.T) {
	valid := dataFrame{Kind: "log", Seq: 1, Text: "hello"}
	payload, err := json.Marshal(valid)
	require.Nil(t, err, fmt.Sprintf("unexpected marshal error: %s", err))

	stream := bytes.NewReader(append(rawFrame([]byte("{")), rawFrame(payload)...))

	var f dataFrame
	_, err = readFrame(stream, &f)
	assert.True(t, errors.Is(err, errBadFrame), fmt.Sprintf("expected %v got %v", errBadFrame, err))

	f = dataFrame{}
	_, err = readFrame(stream, &f)
	require.Nil(t, err, fmt.Sprintf("expected the next frame to be readable, got %s", err))
	assert.Equal(t, valid, f, "expected the frame after a bad one to decode")
}

func TestPumpData(t *testing.T) {
	r, wr, err := os.Pipe()
	require.Nil(t, err, fmt.Sprintf("unexpected pipe error: %s", err))
	defer r.Close()

	w := &worker{
		data:   r,
		dataCh: make(chan dataMsg, 4),
		errCh:  make(chan error, 2),
		done:   make(chan struct{}),
	}
	go w.pumpData()

	valid := dataFrame{Kind: "log", Seq: 1, Text: "hello"}
	payload, err := json.Marshal(valid)
	require.Nil(t, err, fmt.Sprintf("unexpected marshal error: %s", err))

	_, err = wr.Write(rawFrame([]byte("garbage")))
	require.Nil(t, err, fmt.Sprintf("unexpected write error: %s", err))
	_, err = wr.Write(rawFrame(payload))
	require.Nil(t, err, fmt.Sprintf("unexpected write error: %s", err))
	wr.Close()

	select {
	case m := <-w.dataCh:
		assert.Equal(t, valid, m.frame, "expected the valid frame to be delivered")
		assert.Equal(t, len(payload), m.size, "expected the frame size to be delivered")
	case <-time.After(time.Second):
		t.Fatal("expected a frame after the undecodable one")
	}

	select {
	case err := <-w.errCh:
		assert.True(t, errors.Is(err, io.EOF), fmt.Sprintf("expected %v got %v", io.EOF, err))
	case <-time.After(time.Second):
		t.Fatal("expected the pump to report the closed pipe")
	}

	assert.Equal(t, int64(1), w.garbled.Load(), "expected the undecodable frame to be counted")
}

func TestTerminationReason(t *testing.T) {
	cfg := Config{Limits: Limits{CPUSeconds: 5}}

	cases := []struct {
		desc   string
		signal int
		status int
		reason string
	}{
		{
			desc:   "explain a cpu limit",
			signal: int(syscall.SIGXCPU),
			reason: "cpu time limit exceeded (5s)",
		},
		{
			desc:   "explain a file size limit",
			signal: int(syscall.SIGXFSZ),
			reason: "file size limit exceeded",
		},
		{
			desc:   "explain a kill",
			signal: int(syscall.SIGKILL),
			reason: "script killed, most likely by the memory limit",
		},
		{
			desc:   "explain a crash",
			signal: int(syscall.SIGSEGV),
			reason: fmt.Sprintf("script crashed (%s)", syscall.SIGSEGV),
		},
		{
			desc:   "explain another signal",
			signal: int(syscall.SIGTERM),
			reason: fmt.Sprintf("script killed by %s", syscall.SIGTERM),
		},
		{
			desc:   "explain an exit without a result",
			status: 2,
			reason: "script exited without a result (code 2)",
		},
	}

	for _, tc := range cases {
		reason := terminationReason(tc.signal, tc.status, cfg)
		assert.Equal(t, tc.reason, reason, fmt.Sprintf("%s: expected %q got %q", tc.desc, tc.reason, reason))
	}
}
