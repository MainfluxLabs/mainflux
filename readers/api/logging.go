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

func (lm *loggingMiddleware) ListJSONMessages(ctx context.Context, token string, key things.ThingKey, rpm readers.JSONPageMetadata) (_ readers.JSONMessagesPage, err error) {
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

func (lm *loggingMiddleware) ListSenMLMessages(ctx context.Context, token string, key things.ThingKey, rpm readers.SenMLPageMetadata) (_ readers.SenMLMessagesPage, err error) {
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

func (lm *loggingMiddleware) Backup(ctx context.Context, token string) (_ readers.Backup, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method backup took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Backup(ctx, token)
}

func (lm *loggingMiddleware) Restore(ctx context.Context, token string, backup readers.Backup) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method restore took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Restore(ctx, token, backup)
}

func (lm *loggingMiddleware) ExportJSONMessages(ctx context.Context, token string, rpm readers.JSONPageMetadata) (_ readers.JSONMessagesPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method export_json_messages took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ExportJSONMessages(ctx, token, rpm)
}

func (lm *loggingMiddleware) ExportSenMLMessages(ctx context.Context, token string, rpm readers.SenMLPageMetadata) (_ readers.SenMLMessagesPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method export_senml_messages took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ExportSenMLMessages(ctx, token, rpm)
}

func (lm *loggingMiddleware) DeleteJSONMessages(ctx context.Context, token string, rpm readers.JSONPageMetadata) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method delete_json_messages took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.DeleteJSONMessages(ctx, token, rpm)
}

func (lm *loggingMiddleware) DeleteSenMLMessages(ctx context.Context, token string, rpm readers.SenMLPageMetadata) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method delete_senml_messages took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.DeleteSenMLMessages(ctx, token, rpm)
}

func (lm *loggingMiddleware) DeleteAllJSONMessages(ctx context.Context, token string, rpm readers.JSONPageMetadata) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method delete_all_json_messages took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.DeleteAllJSONMessages(ctx, token, rpm)
}

func (lm *loggingMiddleware) DeleteAllSenMLMessages(ctx context.Context, token string, rpm readers.SenMLPageMetadata) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method delete_all_senml_messages took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.DeleteAllSenMLMessages(ctx, token, rpm)
}
