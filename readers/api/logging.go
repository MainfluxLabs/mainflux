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
)

var _ readers.MessageRepository = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger logger.Logger
	svc    readers.MessageRepository
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc readers.MessageRepository, logger logger.Logger) readers.MessageRepository {
	return &loggingMiddleware{
		logger: logger,
		svc:    svc,
	}
}

func (lm *loggingMiddleware) ListJSONMessages(ctx context.Context, rpm readers.JSONMetadata) (page readers.JSONMessagesPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_json_messages took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListJSONMessages(ctx, rpm)
}

func (lm *loggingMiddleware) ListSenMLMessages(ctx context.Context, rpm readers.SenMLMetadata) (page readers.SenMLMessagesPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_senml_messages took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListSenMLMessages(ctx, rpm)
}

func (lm *loggingMiddleware) BackupJSONMessages(ctx context.Context, rpm readers.JSONMetadata) (page readers.JSONMessagesPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method backup_json_messages took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.BackupJSONMessages(ctx, rpm)
}

func (lm *loggingMiddleware) BackupSenMLMessages(ctx context.Context, rpm readers.SenMLMetadata) (page readers.SenMLMessagesPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method backup_senml_messages took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.BackupSenMLMessages(ctx, rpm)
}

func (lm *loggingMiddleware) RestoreJSONMessages(ctx context.Context, messages ...readers.Message) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method restore_json_messages took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RestoreSenMLMessages(ctx, messages...)
}

func (lm *loggingMiddleware) RestoreSenMLMessages(ctx context.Context, messages ...readers.Message) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method restore_senml_messages took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RestoreSenMLMessages(ctx, messages...)
}

func (lm *loggingMiddleware) DeleteJSONMessages(ctx context.Context, rpm readers.JSONMetadata) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method delete_json_messages took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.DeleteJSONMessages(ctx, rpm)
}

func (lm *loggingMiddleware) DeleteSenMLMessages(ctx context.Context, rpm readers.SenMLMetadata) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method delete_senml_messages took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.DeleteSenMLMessages(ctx, rpm)
}
