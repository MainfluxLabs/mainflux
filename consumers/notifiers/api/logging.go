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
	"github.com/MainfluxLabs/mainflux/things"
)

var _ notifiers.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    notifiers.Service
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc notifiers.Service, logger log.Logger) notifiers.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) CreateNotifiers(ctx context.Context, token string, notifiers ...things.Notifier) (response []things.Notifier, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_notifiers for notifiers %s took %s to complete", response, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateNotifiers(ctx, token, notifiers...)
}

func (lm *loggingMiddleware) ListNotifiersByGroup(ctx context.Context, token string, groupID string) (response []things.Notifier, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_notifiers_by_group for id %s took %s to complete", groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListNotifiersByGroup(ctx, token, groupID)
}

func (lm *loggingMiddleware) ViewNotifier(ctx context.Context, token, id string) (response things.Notifier, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_notifier for id %s took %s to complete", id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewNotifier(ctx, token, id)
}

func (lm *loggingMiddleware) UpdateNotifier(ctx context.Context, token string, notifier things.Notifier) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_notifier for id %s took %s to complete", notifier.ID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateNotifier(ctx, token, notifier)
}

func (lm *loggingMiddleware) RemoveNotifiers(ctx context.Context, token, groupID string, id ...string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_notifiers took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveNotifiers(ctx, token, groupID, id...)
}

func (lm *loggingMiddleware) Consume(msg interface{}) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method consume took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Consume(msg)
}
