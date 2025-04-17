package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/rules"
	"github.com/opentracing/opentracing-go"
)

const (
	saveRuleOp          = "save_rule"
	retrieveByIDOp      = "retrieve_by_id"
	retrieveByProfileOp = "retrieve_by_profile"
	retrieveByGroupOp   = "retrieve_by_group"
	updateRuleOp        = "update_rule"
	removeRuleOp        = "remove_rule"
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
	span := createSpan(ctx, rpm.tracer, saveRuleOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.Save(ctx, rules...)
}

func (rpm ruleRepositoryMiddleware) RetrieveByID(ctx context.Context, id string) (rules.Rule, error) {
	span := createSpan(ctx, rpm.tracer, retrieveByIDOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.RetrieveByID(ctx, id)
}

func (rpm ruleRepositoryMiddleware) RetrieveByProfile(ctx context.Context, profileID string, pm apiutil.PageMetadata) (rules.RulesPage, error) {
	span := createSpan(ctx, rpm.tracer, retrieveByProfileOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.RetrieveByProfile(ctx, profileID, pm)
}

func (rpm ruleRepositoryMiddleware) RetrieveByGroup(ctx context.Context, groupID string, pm apiutil.PageMetadata) (rules.RulesPage, error) {
	span := createSpan(ctx, rpm.tracer, retrieveByGroupOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.RetrieveByGroup(ctx, groupID, pm)
}

func (rpm ruleRepositoryMiddleware) Update(ctx context.Context, rule rules.Rule) error {
	span := createSpan(ctx, rpm.tracer, updateRuleOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.Update(ctx, rule)
}

func (rpm ruleRepositoryMiddleware) Remove(ctx context.Context, ids ...string) error {
	span := createSpan(ctx, rpm.tracer, removeRuleOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.Remove(ctx, ids...)
}

func createSpan(ctx context.Context, tracer opentracing.Tracer, opName string) opentracing.Span {
	if parentSpan := opentracing.SpanFromContext(ctx); parentSpan != nil {
		return tracer.StartSpan(
			opName,
			opentracing.ChildOf(parentSpan.Context()),
		)
	}
	return tracer.StartSpan(opName)
}
