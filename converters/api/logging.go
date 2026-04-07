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
	"github.com/MainfluxLabs/mainflux/pkg/domain"
)

var _ converters.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    converters.Service
	auth   domain.AuthClient
}

// LoggingMiddleware adds logging facilities to the adapter.
func LoggingMiddleware(svc converters.Service, logger log.Logger, auth domain.AuthClient) converters.Service {
	return &loggingMiddleware{logger, svc, auth}
}

func (lm *loggingMiddleware) identify(token string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	id, err := lm.auth.Identify(ctx, token)
	if err != nil {
		return ""
	}
	return id.Email
}

func (lm *loggingMiddleware) PublishJSONMessages(ctx context.Context, token string, csvLines [][]string) (err error) {
	defer func(begin time.Time) {
		email := lm.identify(token)
		message := fmt.Sprintf("Method publish_json_messages by user %s took %s to complete", email, time.Since(begin))
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
		email := lm.identify(token)
		message := fmt.Sprintf("Method publish_senml_messages by user %s took %s to complete", email, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.PublishSenMLMessages(ctx, token, csvLines)
}
