// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package tracing contains middlewares that will add spans
// to existing traces.
package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/users"
	opentracing "github.com/opentracing/opentracing-go"
)

const (
	saveOp            = "save"
	updateOp          = "update"
	retrieveByEmailOp = "retrieve_by_email"
	retrieveByIDOp    = "retrieve_by_id"
	retrieveByIDsOp   = "retrieve_by_ids"
	retrieveAllOp     = "retrieve_all"
	updatePasswordOp  = "update_password"
	changeStatusOp    = "change_status"
)

var _ users.UserRepository = (*userRepositoryMiddleware)(nil)

type userRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   users.UserRepository
}

// UserRepositoryMiddleware tracks request and their latency, and adds spans
// to context.
func UserRepositoryMiddleware(repo users.UserRepository, tracer opentracing.Tracer) users.UserRepository {
	return userRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (urm userRepositoryMiddleware) Save(ctx context.Context, user users.User) (string, error) {
	span := createSpan(ctx, urm.tracer, saveOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.Save(ctx, user)
}

func (urm userRepositoryMiddleware) UpdateUser(ctx context.Context, user users.User) error {
	span := createSpan(ctx, urm.tracer, updateOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.UpdateUser(ctx, user)
}

func (urm userRepositoryMiddleware) RetrieveByEmail(ctx context.Context, email string) (users.User, error) {
	span := createSpan(ctx, urm.tracer, retrieveByEmailOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.RetrieveByEmail(ctx, email)
}

func (urm userRepositoryMiddleware) RetrieveByID(ctx context.Context, id string) (users.User, error) {
	span := createSpan(ctx, urm.tracer, retrieveByIDOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.RetrieveByID(ctx, id)
}

func (urm userRepositoryMiddleware) UpdatePassword(ctx context.Context, email, password string) error {
	span := createSpan(ctx, urm.tracer, updatePasswordOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.UpdatePassword(ctx, email, password)
}

func (urm userRepositoryMiddleware) RetrieveByIDs(ctx context.Context, ids []string, pm users.PageMetadata) (users.UserPage, error) {
	span := createSpan(ctx, urm.tracer, retrieveByIDsOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.RetrieveByIDs(ctx, ids, pm)
}

func (urm userRepositoryMiddleware) RetrieveAll(ctx context.Context) ([]users.User, error) {
	span := createSpan(ctx, urm.tracer, retrieveAllOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.RetrieveAll(ctx)
}

func (urm userRepositoryMiddleware) ChangeStatus(ctx context.Context, id, status string) error {
	span := createSpan(ctx, urm.tracer, changeStatusOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.ChangeStatus(ctx, id, status)
}

func createSpan(ctx context.Context, tracer opentracing.Tracer, opName string) opentracing.Span {
	if parentSpan := opentracing.SpanFromContext(ctx); parentSpan != nil {
		return tracer.StartSpan(
			opName,
			opentracing.ChildOf(parentSpan.Context()),
		)
	}
	return tracer.StartSpan(opName)
}
