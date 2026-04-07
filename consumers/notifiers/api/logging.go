// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"fmt"
	"time"

	notifiers "github.com/MainfluxLabs/mainflux/consumers/notifiers"
	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
)

var _ notifiers.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    notifiers.Service
	auth   domain.AuthClient
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc notifiers.Service, logger log.Logger, auth domain.AuthClient) notifiers.Service {
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

func (lm *loggingMiddleware) CreateNotifiers(ctx context.Context, token, groupID string, notifiers ...notifiers.Notifier) (response []notifiers.Notifier, err error) {
	defer func(begin time.Time) {
		email := lm.identify(token)
		message := fmt.Sprintf("Method create_notifiers by user %s, notifiers %v took %s to complete", email, response, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateNotifiers(ctx, token, groupID, notifiers...)
}

func (lm *loggingMiddleware) ListNotifiersByGroup(ctx context.Context, token string, groupID string, pm notifiers.PageMetadata) (_ notifiers.NotifiersPage, err error) {
	defer func(begin time.Time) {
		email := lm.identify(token)
		message := fmt.Sprintf("Method list_notifiers_by_group by user %s, group id %s took %s to complete", email, groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListNotifiersByGroup(ctx, token, groupID, pm)
}

func (lm *loggingMiddleware) ViewNotifier(ctx context.Context, token, id string) (_ notifiers.Notifier, err error) {
	defer func(begin time.Time) {
		email := lm.identify(token)
		message := fmt.Sprintf("Method view_notifier by user %s, notifier id %s took %s to complete", email, id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewNotifier(ctx, token, id)
}

func (lm *loggingMiddleware) UpdateNotifier(ctx context.Context, token string, notifier notifiers.Notifier) (err error) {
	defer func(begin time.Time) {
		email := lm.identify(token)
		message := fmt.Sprintf("Method update_notifier by user %s, notifier id %s took %s to complete", email, notifier.ID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateNotifier(ctx, token, notifier)
}

func (lm *loggingMiddleware) RemoveNotifiers(ctx context.Context, token string, id ...string) (err error) {
	defer func(begin time.Time) {
		email := lm.identify(token)
		message := fmt.Sprintf("Method remove_notifiers by user %s, notifier ids %v took %s to complete", email, id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveNotifiers(ctx, token, id...)
}

func (lm *loggingMiddleware) RemoveNotifiersByGroup(ctx context.Context, groupID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_notifiers_by_group for group id %s took %s to complete", groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveNotifiersByGroup(ctx, groupID)
}

func (lm *loggingMiddleware) Consume(subject string, msg any) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method consume took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Consume(subject, msg)
}
