// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"github.com/MainfluxLabs/mainflux/audit"
	log "github.com/MainfluxLabs/mainflux/logger"
)

var _ audit.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    audit.Service
}

func LoggingMiddleware(svc audit.Service, logger log.Logger) audit.Service {
	return &loggingMiddleware{logger, svc}
}
