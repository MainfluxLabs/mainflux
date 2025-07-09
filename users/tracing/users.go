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
	save            = "save"
	update          = "update"
	retrieveByEmail = "retrieve_by_email"
	retrieveByID    = "retrieve_by_id"
	retrieveByIDs   = "retrieve_by_ids"
	retrieveAll     = "retrieve_all"
	backupAll       = "backup_all"
	updatePassword  = "update_password"
	changeStatus    = "change_status"
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
	span := createSpan(ctx, urm.tracer, save)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.Save(ctx, user)
}

func (urm userRepositoryMiddleware) UpdateUser(ctx context.Context, user users.User) error {
	span := createSpan(ctx, urm.tracer, update)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.UpdateUser(ctx, user)
}

func (urm userRepositoryMiddleware) RetrieveByEmail(ctx context.Context, email string) (users.User, error) {
	span := createSpan(ctx, urm.tracer, retrieveByEmail)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.RetrieveByEmail(ctx, email)
}

func (urm userRepositoryMiddleware) RetrieveByID(ctx context.Context, id string) (users.User, error) {
	span := createSpan(ctx, urm.tracer, retrieveByID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.RetrieveByID(ctx, id)
}

func (urm userRepositoryMiddleware) UpdatePassword(ctx context.Context, email, password string) error {
	span := createSpan(ctx, urm.tracer, updatePassword)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.UpdatePassword(ctx, email, password)
}

func (urm userRepositoryMiddleware) RetrieveByIDs(ctx context.Context, ids []string, pm users.PageMetadata) (users.UserPage, error) {
	span := createSpan(ctx, urm.tracer, retrieveByIDs)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.RetrieveByIDs(ctx, ids, pm)
}

func (urm userRepositoryMiddleware) BackupAll(ctx context.Context) ([]users.User, error) {
	span := createSpan(ctx, urm.tracer, backupAll)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.BackupAll(ctx)
}

func (urm userRepositoryMiddleware) ChangeStatus(ctx context.Context, id, status string) error {
	span := createSpan(ctx, urm.tracer, changeStatus)
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
