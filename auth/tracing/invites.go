// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package tracing contains middlewares that will add spans to existing traces.
package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/auth"
	opentracing "github.com/opentracing/opentracing-go"
)

const (
	save   = "save"
	remove = "remove"
)

var _ auth.InvitesRepository = (*invitesRepositoryMiddleware)(nil)

type invitesRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   auth.InvitesRepository
}

func InvitesRepositoryMiddleware(tracer opentracing.Tracer, repo auth.InvitesRepository) auth.InvitesRepository {
	return invitesRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (irm invitesRepositoryMiddleware) Save(ctx context.Context, invites ...auth.Invite) error {
	span := createSpan(ctx, irm.tracer, save)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.Save(ctx, invites...)
}

func (irm invitesRepositoryMiddleware) RetrieveByID(ctx context.Context, inviteID string) (auth.Invite, error) {
	span := createSpan(ctx, irm.tracer, retrieveByID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.RetrieveByID(ctx, inviteID)
}

func (irm invitesRepositoryMiddleware) Remove(ctx context.Context, inviteID string) error {
	span := createSpan(ctx, irm.tracer, remove)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return irm.repo.Remove(ctx, inviteID)
}
