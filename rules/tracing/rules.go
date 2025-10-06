package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/rules"
	"github.com/opentracing/opentracing-go"
)

const (
	saveRule              = "save_rule"
	retrieveRuleByID      = "retrieve_rule_by_id"
	retrieveRuleByProfile = "retrieve_rule_by_profile"
	retrieveRuleByGroup   = "retrieve_rule_by_group"
	updateRule            = "update_rule"
	removeRule            = "remove_rule"
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

func (rpm ruleRepositoryMiddleware) RetrieveByProfile(ctx context.Context, profileID string, pm apiutil.PageMetadata) (rules.RulesPage, error) {
	span := dbutil.CreateSpan(ctx, rpm.tracer, retrieveRuleByProfile)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.RetrieveByProfile(ctx, profileID, pm)
}

func (rpm ruleRepositoryMiddleware) RetrieveByGroup(ctx context.Context, groupID string, pm apiutil.PageMetadata) (rules.RulesPage, error) {
	span := dbutil.CreateSpan(ctx, rpm.tracer, retrieveRuleByGroup)
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

func (rpm ruleRepositoryMiddleware) Remove(ctx context.Context, ids ...string) error {
	span := dbutil.CreateSpan(ctx, rpm.tracer, removeRule)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.Remove(ctx, ids...)
}
