// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// go:build !test

package api

import (
	"context"
	"fmt"
	"time"

	log "github.com/MainfluxLabs/mainflux/logger"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
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

func (lm *loggingMiddleware) Publish(ctx context.Context, thingKey string, msg protomfx.Message) (err error) {
	defer func(begin time.Time) {
		destProfile := msg.GetProfile()
		if msg.Subtopic != "" {
			destProfile = fmt.Sprintf("%s.%s", destProfile, msg.Subtopic)
		}
		message := fmt.Sprintf("Method publish to %s took %s to complete", destProfile, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Publish(ctx, thingKey, msg)
}

func (lm *loggingMiddleware) Subscribe(ctx context.Context, thingKey, profileID, subtopic string, c *ws.Client) (err error) {
	defer func(begin time.Time) {
		destProfile := profileID
		if subtopic != "" {
			destProfile = fmt.Sprintf("%s.%s", destProfile, subtopic)
		}
		message := fmt.Sprintf("Method subscribe took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Subscribe(ctx, thingKey, profileID, subtopic, c)
}

func (lm *loggingMiddleware) Unsubscribe(ctx context.Context, thingKey, profileID, subtopic string) (err error) {
	defer func(begin time.Time) {
		destProfile := profileID
		if subtopic != "" {
			destProfile = fmt.Sprintf("%s.%s", destProfile, subtopic)
		}
		message := fmt.Sprintf("Method unsubscribe took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Unsubscribe(ctx, thingKey, profileID, subtopic)
}
