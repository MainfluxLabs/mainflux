package api

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/rules"
	"github.com/go-kit/kit/metrics"
)

var _ rules.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     rules.Service
}

// MetricsMiddleware instruments core service by tracking request count and latency.
func MetricsMiddleware(svc rules.Service, counter metrics.Counter, latency metrics.Histogram) rules.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (ms metricsMiddleware) CreateRules(ctx context.Context, token string, rules ...rules.Rule) ([]rules.Rule, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "create_rules").Add(1)
		ms.latency.With("method", "create_rules").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.CreateRules(ctx, token, rules...)
}

func (ms metricsMiddleware) ListRulesByProfile(ctx context.Context, token, profileID string, pm apiutil.PageMetadata) (rules.RulesPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_rules_by_profile").Add(1)
		ms.latency.With("method", "list_rules_by_profile").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListRulesByProfile(ctx, token, profileID, pm)
}

func (ms metricsMiddleware) ListRulesByGroup(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (rules.RulesPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_rules_by_group").Add(1)
		ms.latency.With("method", "list_rules_by_group").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListRulesByGroup(ctx, token, groupID, pm)
}

func (ms metricsMiddleware) ViewRule(ctx context.Context, token, id string) (rules.Rule, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_rule").Add(1)
		ms.latency.With("method", "view_rule").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewRule(ctx, token, id)
}

func (ms metricsMiddleware) UpdateRule(ctx context.Context, token string, rule rules.Rule) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_rule").Add(1)
		ms.latency.With("method", "update_rule").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UpdateRule(ctx, token, rule)
}

func (ms metricsMiddleware) RemoveRules(ctx context.Context, token string, ids ...string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_rules").Add(1)
		ms.latency.With("method", "remove_rules").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveRules(ctx, token, ids...)
}

func (ms metricsMiddleware) Publish(ctx context.Context, message protomfx.Message) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "publish").Add(1)
		ms.latency.With("method", "publish").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Publish(ctx, message)
}
