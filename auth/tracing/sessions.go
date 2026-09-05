// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	opentracing "github.com/opentracing/opentracing-go"
)

const (
	saveSession           = "save_session"
	retrieveSession       = "retrieve_session"
	rotateSession         = "rotate_session"
	revokeSessionFamily   = "revoke_session_family"
	revokeSessionsByUser  = "revoke_sessions_by_user"
	removeExpiredSessions = "remove_expired_sessions"
)

var _ auth.SessionRepository = (*sessionRepositoryMiddleware)(nil)

type sessionRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   auth.SessionRepository
}

func SessionsRepositoryMiddleware(tracer opentracing.Tracer, sr auth.SessionRepository) auth.SessionRepository {
	return sessionRepositoryMiddleware{
		tracer: tracer,
		repo:   sr,
	}
}

func (srm sessionRepositoryMiddleware) Save(ctx context.Context, session auth.Session) error {
	span := dbutil.CreateSpan(ctx, srm.tracer, saveSession)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return srm.repo.Save(ctx, session)
}

func (srm sessionRepositoryMiddleware) Retrieve(ctx context.Context, jti string) (auth.Session, error) {
	span := dbutil.CreateSpan(ctx, srm.tracer, retrieveSession)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return srm.repo.Retrieve(ctx, jti)
}

func (srm sessionRepositoryMiddleware) Rotate(ctx context.Context, jti, nextJTI string, at time.Time) (auth.Session, bool, error) {
	span := dbutil.CreateSpan(ctx, srm.tracer, rotateSession)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return srm.repo.Rotate(ctx, jti, nextJTI, at)
}

func (srm sessionRepositoryMiddleware) RevokeFamily(ctx context.Context, familyID string, at time.Time) error {
	span := dbutil.CreateSpan(ctx, srm.tracer, revokeSessionFamily)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return srm.repo.RevokeFamily(ctx, familyID, at)
}

func (srm sessionRepositoryMiddleware) RevokeByUser(ctx context.Context, userID string, at time.Time) error {
	span := dbutil.CreateSpan(ctx, srm.tracer, revokeSessionsByUser)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return srm.repo.RevokeByUser(ctx, userID, at)
}

func (srm sessionRepositoryMiddleware) RemoveExpired(ctx context.Context, before time.Time) error {
	span := dbutil.CreateSpan(ctx, srm.tracer, removeExpiredSessions)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return srm.repo.RemoveExpired(ctx, before)
}
