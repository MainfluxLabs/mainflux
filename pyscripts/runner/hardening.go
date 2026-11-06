// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build linux

package runner

import (
	"bytes"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/MainfluxLabs/mainflux/logger"
	"golang.org/x/sys/unix"
)

// Linux is the only platform that implements either of these, so they are
// installed over the no-op defaults in runner.go rather than being called
// directly. That keeps the platform-specific code in this one file.
func init() {
	hideProcess = hideProcessLinux
	reapOrphans = reapOrphansLinux
	parentOf = parentOfLinux
}

func hideProcessLinux() error {
	return unix.Prctl(unix.PR_SET_DUMPABLE, 0, 0, 0, 0)
}
func reapOrphansLinux(log logger.Logger) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, unix.SIGCHLD)

	go func() {
		for range sigs {
			for {
				var status unix.WaitStatus
				pid, err := unix.Wait4(-1, &status, unix.WNOHANG, nil)
				if pid <= 0 || err != nil {
					break
				}

				log.Warn(fmt.Sprintf("reaped child pid=%d %s", pid, describeWait(status)))
			}
		}
	}()
}

func describeWait(s unix.WaitStatus) string {
	switch {
	case s.Signaled():
		return fmt.Sprintf("killed by %s", s.Signal())
	case s.Exited():
		return fmt.Sprintf("exited with code %d", s.ExitStatus())
	default:
		return "stopped or continued"
	}
}

func parentOfLinux(pid int) int {
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return -1
	}

	i := bytes.LastIndexByte(b, ')')
	if i < 0 {
		return -1
	}

	fields := strings.Fields(string(b[i+1:]))
	if len(fields) < 2 {
		return -1
	}

	ppid, err := strconv.Atoi(fields[1])
	if err != nil {
		return -1
	}

	return ppid
}
