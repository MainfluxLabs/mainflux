// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"fmt"
	"time"

	"github.com/MainfluxLabs/mainflux/converters"
	log "github.com/MainfluxLabs/mainflux/logger"
)

var _ converters.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    converters.Service
}

// LoggingMiddleware adds logging facilities to the adapter.
func LoggingMiddleware(svc converters.Service, logger log.Logger) converters.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) PublishJSONMessages(ctx context.Context, token string, csvLines [][]string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method publish_json_messages took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.PublishJSONMessages(ctx, token, csvLines)
}

func (lm *loggingMiddleware) PublishSenMLMessages(ctx context.Context, token string, csvLines [][]string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method publish_senml_messages took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.PublishSenMLMessages(ctx, token, csvLines)
}
