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
	"syscall"
)

const maxFrame = 8 << 20

// errBadFrame marks a frame whose payload could not be decoded although the
// stream stayed aligned. It never reaches a caller: only pumpData interprets it.
var errBadFrame = errors.New("undecodable frame")

type jobFrame struct {
	Seq    uint64 `json:"seq"`
	Code   string `json:"code,omitempty"`
	Input  Input  `json:"input"`
	Limits Limits `json:"limits"`
	Check  bool   `json:"check,omitempty"`
}

// ctlFrame travels on the worker's stdout. The forked child has /dev/null on its
// fd 1, so only the worker parent can write here - which is what makes "exit"
// unforgeable by user code.
type ctlFrame struct {
	Kind    string `json:"kind"`
	Seq     uint64 `json:"seq,omitempty"`
	PID     int    `json:"pid,omitempty"`
	Status  int    `json:"status,omitempty"`
	Signal  int    `json:"signal,omitempty"`
	Nonce   string `json:"nonce,omitempty"`
	Loaded  string `json:"loaded,omitempty"`
	Sandbox string `json:"sandbox,omitempty"`
	Hidden  bool   `json:"hidden,omitempty"`
}

// dataFrame travels on fd 3, which user code can write to, so nothing on it is
// trusted to end a run.
//
// Seq names the job a frame belongs to, so a straggler from an earlier run on
// the same worker is recognised. It is not a security control.
type dataFrame struct {
	Kind  string `json:"kind"`
	Seq   uint64 `json:"seq"`
	Nonce string `json:"nonce,omitempty"`
	Text  string `json:"text,omitempty"`
	Value any    `json:"value,omitempty"`
	Error string `json:"error,omitempty"`
}

type dataMsg struct {
	frame dataFrame
	size  int
}

func writeFrame(w io.Writer, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}

	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(b)))

	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}

	_, err = w.Write(b)
	return err
}

// readFrame decodes one frame and reports how many payload bytes it carried, so
// callers can bound total output rather than only the parts they keep.
func readFrame(r io.Reader, v any) (int, error) {
	var hdr [4]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return 0, err
	}

	n := binary.BigEndian.Uint32(hdr[:])
	if n > maxFrame {
		return 0, ErrWorkerFailed
	}

	b := make([]byte, n)
	if _, err := io.ReadFull(r, b); err != nil {
		return 0, err
	}

	// The body was consumed whole, so the stream is still aligned and the
	// reader can skip this frame.
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	if err := dec.Decode(v); err != nil {
		return len(b), errBadFrame
	}

	return len(b), nil
}

// pumpCtl and pumpData feed the worker's channels for its lifetime. Both select
// on done when handing a frame over: closing the pipes frees a pump sitting in
// readFrame, but one already blocked on a send to a worker nobody will read
// again would never notice, leaking a goroutine and its buffered frames.
func (w *worker) pumpCtl() {
	for {
		var f ctlFrame
		if _, err := readFrame(w.ctl, &f); err != nil {
			select {
			case w.errCh <- err:
			default:
			}
			return
		}

		select {
		case w.ctlCh <- f:
		case <-w.done:
			return
		}
	}
}

func (w *worker) pumpData() {
	for {
		var d dataFrame
		n, err := readFrame(w.data, &d)

		// The child can write anything to fd 3, so an unparseable frame is
		// counted and dropped rather than costing the worker.
		if err == errBadFrame {
			w.garbled.Add(1)
			continue
		}

		if err != nil {
			select {
			case w.errCh <- err:
			default:
			}
			return
		}
		select {
		case w.dataCh <- dataMsg{frame: d, size: n}:
		case <-w.done:
			return
		}
	}
}

// terminationReason explains a child that died without producing a result. The
// cases Go itself caused - deadline and output limit - are reported by the
// caller; what reaches here is the kernel's doing.
func terminationReason(sig, code int, cfg Config) string {
	switch syscall.Signal(sig) {
	case syscall.SIGXCPU:
		return fmt.Sprintf("cpu time limit exceeded (%ds)", cfg.Limits.CPUSeconds)
	case syscall.SIGXFSZ:
		return "file size limit exceeded"
	case syscall.SIGKILL:
		return "script killed, most likely by the memory limit"
	case syscall.SIGSEGV, syscall.SIGBUS, syscall.SIGILL, syscall.SIGFPE:
		return fmt.Sprintf("script crashed (%s)", syscall.Signal(sig))
	}

	if sig != 0 {
		return fmt.Sprintf("script killed by %s", syscall.Signal(sig))
	}

	return fmt.Sprintf("script exited without a result (code %d)", code)
}
