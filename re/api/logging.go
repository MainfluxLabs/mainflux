//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

// +build !test

package api

import (
	"context"
	"fmt"
	"time"

	log "github.com/mainflux/mainflux/logger"
	"github.com/mainflux/mainflux/re"
)

var _ re.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    re.Service
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc re.Service, logger log.Logger) re.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) Info(ctx context.Context) (info re.Info, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method info took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.Info(ctx)
}

func (lm *loggingMiddleware) CreateStream(ctx context.Context, token, name, topic, row string) (result string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_stream for %s with topic %s took %s to complete", name, topic, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateStream(ctx, token, name, topic, row)
}

func (lm *loggingMiddleware) UpdateStream(ctx context.Context, token, name, topic, row string) (result string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_stream for %s and topic %s took %s to complete", name, topic, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateStream(ctx, token, name, topic, row)
}

func (lm *loggingMiddleware) ListStreams(ctx context.Context, token string) (streams []string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_streams took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListStreams(ctx, token)
}

func (lm *loggingMiddleware) ViewStream(ctx context.Context, token, name string) (stream re.Stream, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_stream for %s took %s to complete", name, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewStream(ctx, token, name)
}

func (lm *loggingMiddleware) DeleteStream(ctx context.Context, token, name string) (result string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method delete_stream for %s took %s to complete", name, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.DeleteStream(ctx, token, name)
}

func (lm *loggingMiddleware) CreateRule(ctx context.Context, token string, rule re.Rule) (result string, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_rule %s took %s to complete", rule.ID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateRule(ctx, token, rule)
}

func (lm *loggingMiddleware) ListRules(ctx context.Context, token string) (ris []re.RuleInfo, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_rules took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListRules(ctx, token)
}

func (lm *loggingMiddleware) ViewRule(ctx context.Context, token, name string) (rule re.Rule, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_rule for %s took %s to complete", name, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewRule(ctx, token, name)
}
