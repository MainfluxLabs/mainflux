package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	opentracing "github.com/opentracing/opentracing-go"
)

const (
	saveRole        = "save_role"
	retrieveRole    = "retrieve_role"
	retrieveRoles   = "retrieve_roles"
	updateRole      = "update_role"
	removeRole      = "remove_role"
	backupAllRoles  = "backup_all_roles"
	restoreAllRoles = "restore_all_roles"
)

var _ auth.RolesRepository = (*rolesRepositoryMiddleware)(nil)

type rolesRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   auth.RolesRepository
}

// RolesRepositoryMiddleware tracks request and their latency, and adds spans to context.
func RolesRepositoryMiddleware(tracer opentracing.Tracer, rr auth.RolesRepository) auth.RolesRepository {
	return rolesRepositoryMiddleware{
		tracer: tracer,
		repo:   rr,
	}
}

func (rrm rolesRepositoryMiddleware) SaveRole(ctx context.Context, id, role string) error {
	span := dbutil.CreateSpan(ctx, rrm.tracer, saveRole)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rrm.repo.SaveRole(ctx, id, role)
}

func (rrm rolesRepositoryMiddleware) RetrieveRole(ctx context.Context, id string) (string, error) {
	span := dbutil.CreateSpan(ctx, rrm.tracer, retrieveRole)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rrm.repo.RetrieveRole(ctx, id)
}

func (rrm rolesRepositoryMiddleware) UpdateRole(ctx context.Context, id, role string) error {
	span := dbutil.CreateSpan(ctx, rrm.tracer, updateRole)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rrm.repo.UpdateRole(ctx, id, role)
}

func (rrm rolesRepositoryMiddleware) RemoveRole(ctx context.Context, id string) error {
	span := dbutil.CreateSpan(ctx, rrm.tracer, removeRole)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rrm.repo.RemoveRole(ctx, id)
}

func (rrm rolesRepositoryMiddleware) RetrieveRoles(ctx context.Context, ids []string) (map[string]string, error) {
	span := dbutil.CreateSpan(ctx, rrm.tracer, retrieveRoles)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rrm.repo.RetrieveRoles(ctx, ids)
}

func (rrm rolesRepositoryMiddleware) BackupAll(ctx context.Context) ([]auth.UserRole, error) {
	span := dbutil.CreateSpan(ctx, rrm.tracer, backupAllRoles)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rrm.repo.BackupAll(ctx)
}

func (rrm rolesRepositoryMiddleware) RestoreAll(ctx context.Context, urs ...auth.UserRole) error {
	span := dbutil.CreateSpan(ctx, rrm.tracer, restoreAllRoles)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return rrm.repo.RestoreAll(ctx, urs...)
}
