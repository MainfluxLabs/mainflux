package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/users"
	opentracing "github.com/opentracing/opentracing-go"
)

const (
	saveIdentity     = "save_identity"
	retrieveIdentity = "retrieve_identity"
	backupAllIdentities = "backup_all_identities"
)

var _ users.IdentityRepository = (*identityRepositoryMiddleware)(nil)

type identityRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   users.IdentityRepository
}

func IdentityRepositoryMiddleware(repo users.IdentityRepository, tracer opentracing.Tracer) users.IdentityRepository {
	return identityRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (irm identityRepositoryMiddleware) Save(ctx context.Context, identity users.Identity) error {
	span := dbutil.CreateSpan(ctx, irm.tracer, saveIdentity)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.Save(ctx, identity)
}

func (irm identityRepositoryMiddleware) Retrieve(ctx context.Context, provider, providerUserID string) (users.Identity, error) {
	span := dbutil.CreateSpan(ctx, irm.tracer, retrieveIdentity)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.Retrieve(ctx, provider, providerUserID)
}

func (irm identityRepositoryMiddleware) BackupAll(ctx context.Context) ([]users.Identity, error) {
	span := dbutil.CreateSpan(ctx, irm.tracer, backupAllIdentities)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.BackupAll(ctx)
}
