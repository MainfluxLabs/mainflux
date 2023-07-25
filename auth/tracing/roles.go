package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/auth"
	opentracing "github.com/opentracing/opentracing-go"
)

const (
	saveRole     = "save_role"
	retrieveRole = "retrieve_role"
	updateRole   = "update_role"
	removeRole   = "remove_role"
)

var _ auth.RolesRepository = (*rolesRepositoryMiddleware)(nil)

type rolesRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   auth.RolesRepository
}

// OrgRepositoryMiddleware tracks request and their latency, and adds spans to context.
func RolesRepositoryMiddleware(tracer opentracing.Tracer, rr auth.RolesRepository) auth.RolesRepository {
	return rolesRepositoryMiddleware{
		tracer: tracer,
		repo:   rr,
	}
}

func (rrm rolesRepositoryMiddleware) SaveRole(ctx context.Context, id, role string) error {
	span := createSpan(ctx, rrm.tracer, saveRole)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rrm.repo.SaveRole(ctx, id, role)
}

func (rrm rolesRepositoryMiddleware) RetrieveRole(ctx context.Context, id string) (string, error) {
	span := createSpan(ctx, rrm.tracer, retrieveRole)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rrm.repo.RetrieveRole(ctx, id)
}

func (rrm rolesRepositoryMiddleware) UpdateRole(ctx context.Context, id, role string) error {
	span := createSpan(ctx, rrm.tracer, updateRole)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rrm.repo.UpdateRole(ctx, id, role)
}

func (rrm rolesRepositoryMiddleware) RemoveRole(ctx context.Context, id string) error {
	span := createSpan(ctx, rrm.tracer, removeRole)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rrm.repo.RemoveRole(ctx, id)
}
