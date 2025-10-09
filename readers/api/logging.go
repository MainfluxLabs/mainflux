// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"fmt"
	"time"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/MainfluxLabs/mainflux/things"
)

var _ readers.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger logger.Logger
	svc    readers.Service
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc readers.Service, logger logger.Logger) readers.Service {
	return &loggingMiddleware{
		logger: logger,
		svc:    svc,
	}
}

func (lm *loggingMiddleware) ListJSONMessages(ctx context.Context, token string, key things.ThingKey, rpm readers.JSONPageMetadata) (page readers.JSONMessagesPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_json_messages took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListJSONMessages(ctx, token, key, rpm)
}

func (lm *loggingMiddleware) ListSenMLMessages(ctx context.Context, token string, key things.ThingKey, rpm readers.SenMLPageMetadata) (page readers.SenMLMessagesPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_senml_messages took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListSenMLMessages(ctx, token, key, rpm)
}

func (lm *loggingMiddleware) BackupJSONMessages(ctx context.Context, token string, rpm readers.JSONPageMetadata) (page readers.JSONMessagesPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method backup_json_messages took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.BackupJSONMessages(ctx, token, rpm)
}

func (lm *loggingMiddleware) BackupSenMLMessages(ctx context.Context, token string, rpm readers.SenMLPageMetadata) (page readers.SenMLMessagesPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method backup_senml_messages took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.BackupSenMLMessages(ctx, token, rpm)
}

func (lm *loggingMiddleware) RestoreJSONMessages(ctx context.Context, token string, messages ...readers.Message) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method restore_json_messages took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RestoreSenMLMessages(ctx, token, messages...)
}

func (lm *loggingMiddleware) RestoreSenMLMessages(ctx context.Context, token string, messages ...readers.Message) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method restore_senml_messages took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RestoreSenMLMessages(ctx, token, messages...)
}

func (lm *loggingMiddleware) DeleteJSONMessages(ctx context.Context, token string, key things.ThingKey, rpm readers.JSONPageMetadata) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method delete_json_messages took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.DeleteJSONMessages(ctx, token, key, rpm)
}

func (lm *loggingMiddleware) DeleteSenMLMessages(ctx context.Context, token string, key things.ThingKey, rpm readers.SenMLPageMetadata) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method delete_senml_messages took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.DeleteSenMLMessages(ctx, token, key, rpm)
}
