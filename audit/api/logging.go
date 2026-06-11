// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"fmt"
	"time"

	"github.com/MainfluxLabs/mainflux/audit"
	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/authn"
	"github.com/MainfluxLabs/mainflux/pkg/events"
)

var _ audit.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    audit.Service
}

func LoggingMiddleware(svc audit.Service, logger log.Logger) audit.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) RecordEvent(ctx context.Context, e events.Event) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method record_event for operation %s by user %s took %s to complete", e.Action.Operation(), e.JWTUserIdentity.Email, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RecordEvent(ctx, e)
}

func (lm *loggingMiddleware) ListEventsByOrg(ctx context.Context, token string, orgID string, pm audit.PageMetadata) (page audit.EventsPage, err error) {
	defer func(begin time.Time) {
		email := authn.EmailFromToken(token)
		message := fmt.Sprintf("Method list_events_by_org by user %s, org id %s took %s to complete", email, orgID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListEventsByOrg(ctx, token, orgID, pm)
}

func (lm *loggingMiddleware) ListEvents(ctx context.Context, token string, pm audit.PageMetadata) (page audit.EventsPage, err error) {
	defer func(begin time.Time) {
		email := authn.EmailFromToken(token)
		message := fmt.Sprintf("Method list_events by user %s took %s to complete", email, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListEvents(ctx, token, pm)
}
