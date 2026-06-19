package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/shadows"
	"github.com/opentracing/opentracing-go"
)

const (
	upsertShadow          = "upsert_shadow"
	retrieveShadowByThing = "retrieve_shadow_by_thing"
	removeShadow          = "remove_shadow"
)

var _ shadows.ShadowRepository = (*shadowRepositoryMiddleware)(nil)

type shadowRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   shadows.ShadowRepository
}

// ShadowRepositoryMiddleware tracks request and their latency, and adds spans to context.
func ShadowRepositoryMiddleware(tracer opentracing.Tracer, repo shadows.ShadowRepository) shadows.ShadowRepository {
	return shadowRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (srm shadowRepositoryMiddleware) Upsert(ctx context.Context, shadow shadows.Shadow) (shadows.Shadow, error) {
	span := dbutil.CreateSpan(ctx, srm.tracer, upsertShadow)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return srm.repo.Upsert(ctx, shadow)
}

func (srm shadowRepositoryMiddleware) RetrieveByThing(ctx context.Context, thingID string) (shadows.Shadow, error) {
	span := dbutil.CreateSpan(ctx, srm.tracer, retrieveShadowByThing)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return srm.repo.RetrieveByThing(ctx, thingID)
}

func (srm shadowRepositoryMiddleware) Remove(ctx context.Context, thingID string) error {
	span := dbutil.CreateSpan(ctx, srm.tracer, removeShadow)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return srm.repo.Remove(ctx, thingID)
}
