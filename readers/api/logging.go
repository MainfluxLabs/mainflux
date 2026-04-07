// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"fmt"
	"time"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/readers"
)

var _ readers.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger logger.Logger
	svc    readers.Service
	auth   domain.AuthClient
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc readers.Service, logger logger.Logger, auth domain.AuthClient) readers.Service {
	return &loggingMiddleware{
		logger: logger,
		svc:    svc,
		auth:   auth,
	}
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

func (lm *loggingMiddleware) ListJSONMessages(ctx context.Context, token string, key domain.ThingKey, rpm readers.JSONPageMetadata) (_ readers.JSONMessagesPage, err error) {
	defer func(begin time.Time) {
		email := lm.identify(token)
		message := fmt.Sprintf("Method list_json_messages by user %s took %s to complete", email, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListJSONMessages(ctx, token, key, rpm)
}

func (lm *loggingMiddleware) ListSenMLMessages(ctx context.Context, token string, key domain.ThingKey, rpm readers.SenMLPageMetadata) (_ readers.SenMLMessagesPage, err error) {
	defer func(begin time.Time) {
		email := lm.identify(token)
		message := fmt.Sprintf("Method list_senml_messages by user %s took %s to complete", email, time.Since(begin))
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
		email := lm.identify(token)
		message := fmt.Sprintf("Method backup by user %s took %s to complete", email, time.Since(begin))
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
		email := lm.identify(token)
		message := fmt.Sprintf("Method restore by user %s took %s to complete", email, time.Since(begin))
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
		email := lm.identify(token)
		message := fmt.Sprintf("Method export_json_messages by user %s took %s to complete", email, time.Since(begin))
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
		email := lm.identify(token)
		message := fmt.Sprintf("Method export_senml_messages by user %s took %s to complete", email, time.Since(begin))
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
		email := lm.identify(token)
		message := fmt.Sprintf("Method delete_json_messages by user %s took %s to complete", email, time.Since(begin))
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
		email := lm.identify(token)
		message := fmt.Sprintf("Method delete_senml_messages by user %s took %s to complete", email, time.Since(begin))
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
		email := lm.identify(token)
		message := fmt.Sprintf("Method delete_all_json_messages by user %s took %s to complete", email, time.Since(begin))
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
		email := lm.identify(token)
		message := fmt.Sprintf("Method delete_all_senml_messages by user %s took %s to complete", email, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.DeleteAllSenMLMessages(ctx, token, rpm)
}
