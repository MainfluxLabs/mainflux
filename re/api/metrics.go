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
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/mainflux/mainflux/re"
)

var _ re.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     re.Service
}

// MetricsMiddleware instruments core service by tracking request count and
// latency.
func MetricsMiddleware(svc re.Service, counter metrics.Counter, latency metrics.Histogram) re.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (ms *metricsMiddleware) Info(ctx context.Context) (info re.Info, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "info").Add(1)
		ms.latency.With("method", "info").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Info(ctx)
}

func (ms *metricsMiddleware) CreateStream(ctx context.Context, token, name, topic, subtopic, row string, update bool) (result string, err error) {
	method := "create_stream"
	if update {
		method = "update_stream"
	}
	defer func(begin time.Time) {
		ms.counter.With("method", method).Add(1)
		ms.latency.With("method", method).Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.CreateStream(ctx, token, name, topic, subtopic, row, update)
}

func (ms *metricsMiddleware) ListStreams(ctx context.Context, token string) (streams []string, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_streams").Add(1)
		ms.latency.With("method", "list_streams").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListStreams(ctx, token)
}

func (ms *metricsMiddleware) ViewStream(ctx context.Context, token, id string) (stream re.Stream, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_stream").Add(1)
		ms.latency.With("method", "view_stream").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewStream(ctx, token, id)
}

func (ms *metricsMiddleware) Delete(ctx context.Context, token string, id string, kind string) (result string, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "delete_"+kind).Add(1)
		ms.latency.With("method", "delete_"+kind).Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Delete(ctx, token, id, kind)
}

func (ms *metricsMiddleware) CreateRule(ctx context.Context, token string, rule re.Rule, update bool) (string, error) {
	method := "create_rule"
	if update {
		method = "update_rule"
	}
	defer func(begin time.Time) {
		ms.counter.With("method", method).Add(1)
		ms.latency.With("method", method).Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.CreateRule(ctx, token, rule, update)
}

func (ms *metricsMiddleware) ListRules(ctx context.Context, token string) ([]re.RuleInfo, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_rules").Add(1)
		ms.latency.With("method", "list_rules").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListRules(ctx, token)
}

func (ms *metricsMiddleware) ViewRule(ctx context.Context, token, id string) (rule re.Rule, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_rule").Add(1)
		ms.latency.With("method", "view_rule").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewRule(ctx, token, id)
}

func (ms *metricsMiddleware) GetRuleStatus(ctx context.Context, token, name string) (map[string]interface{}, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "get_rule_status").Add(1)
		ms.latency.With("method", "get_rule_status").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.GetRuleStatus(ctx, token, name)
}

func (ms *metricsMiddleware) ControlRule(ctx context.Context, token, name, action string) (string, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "get_rule_status").Add(1)
		ms.latency.With("method", "get_rule_status").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ControlRule(ctx, token, name, action)
}
