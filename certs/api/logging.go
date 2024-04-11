// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"fmt"
	"time"

	"github.com/MainfluxLabs/mainflux/certs"
	log "github.com/MainfluxLabs/mainflux/logger"
)

var _ certs.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    certs.Service
}

// NewLoggingMiddleware adds logging facilities to the core service.
func NewLoggingMiddleware(svc certs.Service, logger log.Logger) certs.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) IssueCert(ctx context.Context, token, thingID, ttl string, keyBits int, keyType string) (c certs.Cert, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method issue_cert for thing: %s took %s to complete", thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.IssueCert(ctx, token, thingID, ttl, keyBits, keyType)
}

func (lm *loggingMiddleware) ListCerts(ctx context.Context, token, thingID string, offset, limit uint64) (cp certs.Page, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_certs for thing id: %s took %s to complete", thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListCerts(ctx, token, thingID, offset, limit)
}

func (lm *loggingMiddleware) ListSerials(ctx context.Context, token, thingID string, offset, limit uint64) (cp certs.Page, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_serials for thing id: %s took %s to complete", thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListSerials(ctx, token, thingID, offset, limit)
}

func (lm *loggingMiddleware) ViewCert(ctx context.Context, token, serialID string) (c certs.Cert, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_cert for serial id %s took %s to complete", serialID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewCert(ctx, token, serialID)
}

func (lm *loggingMiddleware) RevokeCert(ctx context.Context, token, thingID string) (c certs.Revoke, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method revoke_cert for thing: %s took %s to complete", thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RevokeCert(ctx, token, thingID)
}
