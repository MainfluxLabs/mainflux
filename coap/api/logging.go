// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"fmt"
	"time"

	"github.com/MainfluxLabs/mainflux/coap"
	log "github.com/MainfluxLabs/mainflux/logger"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

var _ coap.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    coap.Service
}

// LoggingMiddleware adds logging facilities to the adapter.
func LoggingMiddleware(svc coap.Service, logger log.Logger) coap.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) Publish(ctx context.Context, key string, msg protomfx.Message) (err error) {
	defer func(begin time.Time) {
		dest := ""
		if msg.Subtopic != "" {
			dest = fmt.Sprintf("to %s", msg.Subtopic)
		}
		message := fmt.Sprintf("Method publish %s took %s to complete", dest, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Publish(ctx, key, msg)
}

func (lm *loggingMiddleware) Subscribe(ctx context.Context, key, subtopic string, c coap.Client) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method subscribe for client %s took %s to complete", c.Token(), time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Subscribe(ctx, key, subtopic, c)
}

func (lm *loggingMiddleware) Unsubscribe(ctx context.Context, key, subtopic, token string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method unsubscribe for the client %s took %s to complete", token, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Unsubscribe(ctx, key, subtopic, token)

}
