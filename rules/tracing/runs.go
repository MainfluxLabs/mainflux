package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/rules"
	"github.com/opentracing/opentracing-go"
)

const saveRuns = "save_runs"

func (rpm ruleRepositoryMiddleware) SaveRuns(ctx context.Context, runs ...rules.RuleRun) ([]rules.RuleRun, error) {
	span := dbutil.CreateSpan(ctx, rpm.tracer, saveRuns)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rpm.repo.SaveRuns(ctx, runs...)
}
