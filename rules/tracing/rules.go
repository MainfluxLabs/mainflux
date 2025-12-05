package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/rules"
	"github.com/opentracing/opentracing-go"
)

const (
	saveRule               = "save_rule"
	retrieveRuleByID       = "retrieve_rule_by_id"
	retrieveRulesByThing   = "retrieve_rules_by_thing"
	retrieveRulesByGroup   = "retrieve_rules_by_group"
	retrieveThingIDsByRule = "retrieve_thing_ids_by_rule"
	updateRule             = "update_rule"
	removeRules            = "remove_rules"
	removeRulesByGroup     = "remove_rules_by_group"
	assignRules            = "assign_rules"
	unassignRules          = "unassign_rules"
	unassignRulesByThing   = "unassign_rules_by_thing"
)

var (
	_ rules.RuleRepository = (*ruleRepositoryMiddleware)(nil)
)

type ruleRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   rules.RuleRepository
}

// RuleRepositoryMiddleware tracks request and their latency, and adds spans to context.
func RuleRepositoryMiddleware(tracer opentracing.Tracer, repo rules.RuleRepository) rules.RuleRepository {
	return ruleRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (rpm ruleRepositoryMiddleware) Save(ctx context.Context, rules ...rules.Rule) ([]rules.Rule, error) {
	span := dbutil.CreateSpan(ctx, rpm.tracer, saveRule)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.Save(ctx, rules...)
}

func (rpm ruleRepositoryMiddleware) RetrieveByID(ctx context.Context, id string) (rules.Rule, error) {
	span := dbutil.CreateSpan(ctx, rpm.tracer, retrieveRuleByID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.RetrieveByID(ctx, id)
}

func (rpm ruleRepositoryMiddleware) RetrieveByThing(ctx context.Context, thingID string, pm apiutil.PageMetadata) (rules.RulesPage, error) {
	span := dbutil.CreateSpan(ctx, rpm.tracer, retrieveRulesByThing)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.RetrieveByThing(ctx, thingID, pm)
}

func (rpm ruleRepositoryMiddleware) RetrieveByGroup(ctx context.Context, groupID string, pm apiutil.PageMetadata) (rules.RulesPage, error) {
	span := dbutil.CreateSpan(ctx, rpm.tracer, retrieveRulesByGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.RetrieveByGroup(ctx, groupID, pm)
}

func (rpm ruleRepositoryMiddleware) RetrieveThingIDsByRule(ctx context.Context, ruleID string) ([]string, error) {
	span := dbutil.CreateSpan(ctx, rpm.tracer, retrieveThingIDsByRule)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.RetrieveThingIDsByRule(ctx, ruleID)
}

func (rpm ruleRepositoryMiddleware) Update(ctx context.Context, rule rules.Rule) error {
	span := dbutil.CreateSpan(ctx, rpm.tracer, updateRule)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.Update(ctx, rule)
}

func (rpm ruleRepositoryMiddleware) Remove(ctx context.Context, ids ...string) error {
	span := dbutil.CreateSpan(ctx, rpm.tracer, removeRules)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.Remove(ctx, ids...)
}

func (rpm ruleRepositoryMiddleware) RemoveByGroup(ctx context.Context, groupID string) error {
	span := dbutil.CreateSpan(ctx, rpm.tracer, removeRulesByGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.RemoveByGroup(ctx, groupID)
}

func (rpm ruleRepositoryMiddleware) Assign(ctx context.Context, thingID string, ruleIDs ...string) error {
	span := dbutil.CreateSpan(ctx, rpm.tracer, assignRules)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.Assign(ctx, thingID, ruleIDs...)
}

func (rpm ruleRepositoryMiddleware) Unassign(ctx context.Context, thingID string, ruleIDs ...string) error {
	span := dbutil.CreateSpan(ctx, rpm.tracer, unassignRules)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.Unassign(ctx, thingID, ruleIDs...)
}

func (rpm ruleRepositoryMiddleware) UnassignByThing(ctx context.Context, thingID string) error {
	span := dbutil.CreateSpan(ctx, rpm.tracer, unassignRulesByThing)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.UnassignByThing(ctx, thingID)
}
