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

func (lm *loggingMiddleware) ListJSONMessages(rpm readers.JSONMetadata) (page readers.MessagesPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_json_messages took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListJSONMessages(rpm)
}

func (lm *loggingMiddleware) ListSenMLMessages(rpm readers.SenMLMetadata) (page readers.MessagesPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_senml_messages took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListSenMLMessages(rpm)
}

func (lm *loggingMiddleware) BackupJSONMessages(rpm readers.JSONMetadata) (page readers.MessagesPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method backup_json_messages took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.BackupJSONMessages(rpm)
}

func (lm *loggingMiddleware) BackupSenMLMessages(rpm readers.SenMLMetadata) (page readers.MessagesPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method backup_senml_messages took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.BackupSenMLMessages(rpm)
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

	return lm.svc.RestoreSenMLMessageS(ctx, messages...)
}

func (lm *loggingMiddleware) RestoreSenMLMessageS(ctx context.Context, messages ...readers.Message) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method restore_senml_messages took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RestoreSenMLMessageS(ctx, messages...)
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
		message := fmt.Sprintf("Method delete_messages took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.DeleteSenMLMessages(ctx, rpm)
}
