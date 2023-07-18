package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/auth"
	opentracing "github.com/opentracing/opentracing-go"
)

const saveRole = "save_role"

var _ auth.RoleRepository = (*roleRepositoryMiddleware)(nil)

type roleRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   auth.RoleRepository
}

// OrgRepositoryMiddleware tracks request and their latency, and adds spans to context.
func RoleRepositoryMiddleware(tracer opentracing.Tracer, rr auth.RoleRepository) auth.RoleRepository {
	return roleRepositoryMiddleware{
		tracer: tracer,
		repo:   rr,
	}
}

func (rrm roleRepositoryMiddleware) SaveRole(ctx context.Context, id, role string) error {
	span := createSpan(ctx, rrm.tracer, saveRole)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rrm.repo.SaveRole(ctx, id, role)
}
