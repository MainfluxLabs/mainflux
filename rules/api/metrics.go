package api

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
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

func (ms metricsMiddleware) CreateRules(ctx context.Context, token, groupID string, rules ...rules.Rule) ([]rules.Rule, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "create_rules").Add(1)
		ms.latency.With("method", "create_rules").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.CreateRules(ctx, token, groupID, rules...)
}

func (ms metricsMiddleware) ListRulesByThing(ctx context.Context, token, thingID string, pm apiutil.PageMetadata) (rules.RulesPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_rules_by_thing").Add(1)
		ms.latency.With("method", "list_rules_by_thing").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListRulesByThing(ctx, token, thingID, pm)
}

func (ms metricsMiddleware) ListRulesByGroup(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (rules.RulesPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_rules_by_group").Add(1)
		ms.latency.With("method", "list_rules_by_group").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListRulesByGroup(ctx, token, groupID, pm)
}

func (ms metricsMiddleware) ListThingIDsByRule(ctx context.Context, token, ruleID string) ([]string, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_thing_ids_by_rule").Add(1)
		ms.latency.With("method", "list_thing_ids_by_rule").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListThingIDsByRule(ctx, token, ruleID)
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

func (ms metricsMiddleware) RemoveRulesByGroup(ctx context.Context, groupID string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_rules_by_group").Add(1)
		ms.latency.With("method", "remove_rules_by_group").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveRulesByGroup(ctx, groupID)
}

func (ms metricsMiddleware) AssignRules(ctx context.Context, token, thingID string, ruleIDs ...string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "assign_rules").Add(1)
		ms.latency.With("method", "assign_rules").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.AssignRules(ctx, token, thingID, ruleIDs...)
}

func (ms metricsMiddleware) UnassignRules(ctx context.Context, token, thingID string, ruleIDs ...string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "unassign_rules").Add(1)
		ms.latency.With("method", "unassign_rules").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UnassignRules(ctx, token, thingID, ruleIDs...)
}

func (ms metricsMiddleware) UnassignRulesByThing(ctx context.Context, thingID string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "unassign_rules_by_thing").Add(1)
		ms.latency.With("method", "unassign_rules_by_thing").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UnassignRulesByThing(ctx, thingID)
}

func (ms metricsMiddleware) Consume(message any) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "consume").Add(1)
		ms.latency.With("method", "consume").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Consume(message)
}

func (ms metricsMiddleware) CreateScripts(ctx context.Context, token, groupID string, scripts ...rules.LuaScript) ([]rules.LuaScript, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "create_scripts").Add(1)
		ms.latency.With("method", "create_scripts").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.CreateScripts(ctx, token, groupID, scripts...)
}

func (ms metricsMiddleware) ListScriptsByThing(ctx context.Context, token, thingID string, pm apiutil.PageMetadata) (rules.LuaScriptsPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_scripts_by_thing").Add(1)
		ms.latency.With("method", "list_scripts_by_thing").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListScriptsByThing(ctx, token, thingID, pm)
}

func (ms metricsMiddleware) ListScriptsByGroup(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (rules.LuaScriptsPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_scripts_by_group").Add(1)
		ms.latency.With("method", "list_scripts_by_group").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListScriptsByGroup(ctx, token, groupID, pm)
}

func (ms metricsMiddleware) ListThingIDsByScript(ctx context.Context, token, scriptID string) ([]string, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_thing_ids_by_script").Add(1)
		ms.latency.With("method", "list_thing_ids_by_script").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListThingIDsByScript(ctx, token, scriptID)
}

func (ms metricsMiddleware) ViewScript(ctx context.Context, token, id string) (rules.LuaScript, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_script").Add(1)
		ms.latency.With("method", "view_script").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewScript(ctx, token, id)
}

func (ms metricsMiddleware) UpdateScript(ctx context.Context, token string, script rules.LuaScript) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_script").Add(1)
		ms.latency.With("method", "update_script").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UpdateScript(ctx, token, script)
}

func (ms metricsMiddleware) RemoveScripts(ctx context.Context, token string, ids ...string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_scripts").Add(1)
		ms.latency.With("method", "remove_scripts").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveScripts(ctx, token, ids...)
}

func (ms metricsMiddleware) AssignScripts(ctx context.Context, token, thingID string, scriptIDs ...string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "assign_scripts").Add(1)
		ms.latency.With("method", "assign_scripts").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.AssignScripts(ctx, token, thingID, scriptIDs...)
}

func (ms metricsMiddleware) UnassignScripts(ctx context.Context, token, thingID string, scriptIDs ...string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "unassign_scripts").Add(1)
		ms.latency.With("method", "unassign_scripts").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UnassignScripts(ctx, token, thingID, scriptIDs...)
}

func (ms metricsMiddleware) ListScriptRunsByThing(ctx context.Context, token, thingID string, pm apiutil.PageMetadata) (rules.ScriptRunsPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_script_runs_by_thing").Add(1)
		ms.latency.With("method", "list_script_runs_by_thing").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListScriptRunsByThing(ctx, token, thingID, pm)
}

func (ms metricsMiddleware) RemoveScriptRuns(ctx context.Context, token string, ids ...string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_script_runs").Add(1)
		ms.latency.With("method", "remove_script_runs").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveScriptRuns(ctx, token, ids...)
}
