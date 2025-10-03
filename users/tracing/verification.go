package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/users"
	opentracing "github.com/opentracing/opentracing-go"
)

const (
	saveVerification            = "save_email_verification"
	removeVerification          = "remove_email_verification"
	retrieveVerificationByToken = "retrieve_verification_by_token"
)

type verificationRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   users.EmailVerificationRepository
}

var _ users.EmailVerificationRepository = (*verificationRepositoryMiddleware)(nil)

func VerificationRepositoryMiddleware(repo users.EmailVerificationRepository, tracer opentracing.Tracer) users.EmailVerificationRepository {
	return verificationRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (evrm verificationRepositoryMiddleware) Save(ctx context.Context, verification users.EmailVerification) (string, error) {
	span := dbutil.CreateSpan(ctx, evrm.tracer, saveVerification)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return evrm.repo.Save(ctx, verification)
}

func (evrm verificationRepositoryMiddleware) RetrieveByToken(ctx context.Context, confirmationToken string) (users.EmailVerification, error) {
	span := dbutil.CreateSpan(ctx, evrm.tracer, retrieveVerificationByToken)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return evrm.repo.RetrieveByToken(ctx, confirmationToken)
}

func (evrm verificationRepositoryMiddleware) Remove(ctx context.Context, confirmationToken string) error {
	span := dbutil.CreateSpan(ctx, evrm.tracer, removeVerification)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return evrm.repo.Remove(ctx, confirmationToken)
}
