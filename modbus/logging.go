// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package modbus

import (
	"context"
	"fmt"
	"time"

	"github.com/MainfluxLabs/mainflux/logger"
)

var _ Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger logger.Logger
	svc    Service
}

// LoggingMiddleware adds logging to Modbus service.
func LoggingMiddleware(svc Service, logger logger.Logger) Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) StartPolling(ctx context.Context) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method start_polling took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.StartPolling(ctx)
}
