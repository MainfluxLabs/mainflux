// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"fmt"
	"time"

	"github.com/MainfluxLabs/mainflux/consumers"
	log "github.com/MainfluxLabs/mainflux/logger"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

var _ consumers.MessageConsumer = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger   log.Logger
	consumer consumers.MessageConsumer
}

// LoggingMiddleware adds logging facilities to the adapter.
func LoggingMiddleware(consumer consumers.MessageConsumer, logger log.Logger) consumers.MessageConsumer {
	return &loggingMiddleware{
		logger:   logger,
		consumer: consumer,
	}
}

func (lm *loggingMiddleware) ConsumeMessage(subject string, msg protomfx.Message) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method consume_message took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.consumer.ConsumeMessage(subject, msg)
}
