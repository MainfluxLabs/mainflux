package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/rules"
	"github.com/opentracing/opentracing-go"
)

const (
	saveRuns            = "save_runs"
	retrieveRunByID     = "retrieve_run_by_id"
	retrieveRunsByThing = "retrieve_runs_by_thing"
	retrieveRunsByRule  = "retrieve_runs_by_rule"
	removeRuns          = "remove_runs"
)

func (rpm ruleRepositoryMiddleware) SaveRuns(ctx context.Context, runs ...rules.RuleRun) ([]rules.RuleRun, error) {
	span := dbutil.CreateSpan(ctx, rpm.tracer, saveRuns)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.SaveRuns(ctx, runs...)
}

func (rpm ruleRepositoryMiddleware) RetrieveRunByID(ctx context.Context, id string) (rules.RuleRun, error) {
	span := dbutil.CreateSpan(ctx, rpm.tracer, retrieveRunByID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.RetrieveRunByID(ctx, id)
}

func (rpm ruleRepositoryMiddleware) RetrieveRunsByThing(ctx context.Context, thingID string, pm rules.PageMetadata) (rules.RuleRunsPage, error) {
	span := dbutil.CreateSpan(ctx, rpm.tracer, retrieveRunsByThing)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.RetrieveRunsByThing(ctx, thingID, pm)
}

func (rpm ruleRepositoryMiddleware) RetrieveRunsByRule(ctx context.Context, ruleID string, pm rules.PageMetadata) (rules.RuleRunsPage, error) {
	span := dbutil.CreateSpan(ctx, rpm.tracer, retrieveRunsByRule)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.RetrieveRunsByRule(ctx, ruleID, pm)
}

func (rpm ruleRepositoryMiddleware) RemoveRuns(ctx context.Context, ids ...string) error {
	span := dbutil.CreateSpan(ctx, rpm.tracer, removeRuns)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.RemoveRuns(ctx, ids...)
}
