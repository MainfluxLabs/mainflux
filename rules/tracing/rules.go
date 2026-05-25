package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/rules"
	"github.com/opentracing/opentracing-go"
)

const (
	saveRule               = "save_rule"
	retrieveRuleByID       = "retrieve_rule_by_id"
	retrieveRulesByThing   = "retrieve_rules_by_thing"
	retrieveRulesByGroup   = "retrieve_rules_by_group"
	updateRule             = "update_rule"
	assignThings           = "assign_things"
	unassignThings         = "unassign_things"
	removeRules            = "remove_rules"
	removeRulesByGroup     = "remove_rules_by_group"
	unassignRulesFromThing = "unassign_rules_from_thing"
)

var (
	_ rules.Repository = (*ruleRepositoryMiddleware)(nil)
)

type ruleRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   rules.Repository
}

// RuleRepositoryMiddleware tracks request and their latency, and adds spans to context.
func RuleRepositoryMiddleware(tracer opentracing.Tracer, repo rules.Repository) rules.Repository {
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

func (rpm ruleRepositoryMiddleware) RetrieveByThing(ctx context.Context, thingID string, pm rules.PageMetadata) (rules.RulesPage, error) {
	span := dbutil.CreateSpan(ctx, rpm.tracer, retrieveRulesByThing)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.RetrieveByThing(ctx, thingID, pm)
}

func (rpm ruleRepositoryMiddleware) RetrieveByGroup(ctx context.Context, groupID string, pm rules.PageMetadata) (rules.RulesPage, error) {
	span := dbutil.CreateSpan(ctx, rpm.tracer, retrieveRulesByGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.RetrieveByGroup(ctx, groupID, pm)
}

func (rpm ruleRepositoryMiddleware) Update(ctx context.Context, rule rules.Rule) error {
	span := dbutil.CreateSpan(ctx, rpm.tracer, updateRule)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.Update(ctx, rule)
}

func (rpm ruleRepositoryMiddleware) AssignThings(ctx context.Context, ruleID string, thingIDs ...string) error {
	span := dbutil.CreateSpan(ctx, rpm.tracer, assignThings)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.AssignThings(ctx, ruleID, thingIDs...)
}

func (rpm ruleRepositoryMiddleware) UnassignThings(ctx context.Context, ruleID string, thingIDs ...string) error {
	span := dbutil.CreateSpan(ctx, rpm.tracer, unassignThings)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.UnassignThings(ctx, ruleID, thingIDs...)
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

func (rpm ruleRepositoryMiddleware) UnassignRulesFromThing(ctx context.Context, thingID string) error {
	span := dbutil.CreateSpan(ctx, rpm.tracer, unassignRulesFromThing)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.UnassignRulesFromThing(ctx, thingID)
}
