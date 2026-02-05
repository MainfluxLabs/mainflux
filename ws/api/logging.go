// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"fmt"
	"time"

	log "github.com/MainfluxLabs/mainflux/logger"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/MainfluxLabs/mainflux/ws"
)

var _ ws.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    ws.Service
}

// LoggingMiddleware adds logging facilities to the adapter
func LoggingMiddleware(svc ws.Service, logger log.Logger) ws.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) Publish(ctx context.Context, key things.ThingKey, msg protomfx.Message) (err error) {
	defer func(begin time.Time) {
		dest := ""
		if msg.Subtopic != "" {
			dest = fmt.Sprintf("to subtopic %s", msg.Subtopic)
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

func (lm *loggingMiddleware) Subscribe(ctx context.Context, key things.ThingKey, subtopic string, c *ws.Client) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method subscribe for subtopic %s took %s to complete", subtopic, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Subscribe(ctx, key, subtopic, c)
}

func (lm *loggingMiddleware) Unsubscribe(ctx context.Context, key things.ThingKey, subtopic string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method unsubscribe for subtopic %s took %s to complete", subtopic, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Unsubscribe(ctx, key, subtopic)
}
