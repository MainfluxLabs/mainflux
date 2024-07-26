// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package servers

import (
	"time"
)

type Config struct {
	ServerName   string
	ServerCert   string
	ServerKey    string
	Port         string
	StopWaitTime time.Duration
}
