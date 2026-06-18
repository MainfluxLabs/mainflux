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
	"github.com/MainfluxLabs/mainflux/pkg/authn"
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

func (lm *loggingMiddleware) PublishSenMLMessagesFromJSON(ctx context.Context, token string, records []map[string]any) (err error) {
	defer func(begin time.Time) {
		email := authn.EmailFromToken(token)
		message := fmt.Sprintf("Method publish_senml_messages_from_json by user %s took %s to complete", email, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.PublishSenMLMessagesFromJSON(ctx, token, records)
}

func (lm *loggingMiddleware) PublishJSONMessagesFromJSON(ctx context.Context, token string, records []map[string]any) (err error) {
	defer func(begin time.Time) {
		email := authn.EmailFromToken(token)
		message := fmt.Sprintf("Method publish_json_messages_from_json by user %s took %s to complete", email, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.PublishJSONMessagesFromJSON(ctx, token, records)
}

func (lm *loggingMiddleware) PublishJSONMessagesFromCSV(ctx context.Context, token string, csvLines [][]string) (err error) {
	defer func(begin time.Time) {
		email := authn.EmailFromToken(token)
		message := fmt.Sprintf("Method publish_json_messages_from_csv by user %s took %s to complete", email, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.PublishJSONMessagesFromCSV(ctx, token, csvLines)
}

func (lm *loggingMiddleware) PublishSenMLMessagesFromCSV(ctx context.Context, token string, csvLines [][]string) (err error) {
	defer func(begin time.Time) {
		email := authn.EmailFromToken(token)
		message := fmt.Sprintf("Method publish_senml_messages_from_csv by user %s took %s to complete", email, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.PublishSenMLMessagesFromCSV(ctx, token, csvLines)
}
