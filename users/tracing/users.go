// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package tracing contains middlewares that will add spans
// to existing traces.
package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/users"
	opentracing "github.com/opentracing/opentracing-go"
)

const (
	saveUser            = "save_user"
	updateUser          = "update_user"
	retrieveUserByEmail = "retrieve_user_by_email"
	retrieveUserByID    = "retrieve_user_by_id"
	retrieveUsersByIDs  = "retrieve_users_by_ids"
	backupAllUsers      = "backup_all_users"
	updateUserPassword  = "update_user_password"
	changeUserStatus    = "change_user_status"
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
	span := dbutil.CreateSpan(ctx, urm.tracer, saveUser)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.Save(ctx, user)
}

func (urm userRepositoryMiddleware) Update(ctx context.Context, user users.User) error {
	span := dbutil.CreateSpan(ctx, urm.tracer, updateUser)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.Update(ctx, user)
}

func (urm userRepositoryMiddleware) UpdateUser(ctx context.Context, user users.User) error {
	span := dbutil.CreateSpan(ctx, urm.tracer, updateUser)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.UpdateUser(ctx, user)
}

func (urm userRepositoryMiddleware) RetrieveByEmail(ctx context.Context, email string) (users.User, error) {
	span := dbutil.CreateSpan(ctx, urm.tracer, retrieveUserByEmail)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.RetrieveByEmail(ctx, email)
}

func (urm userRepositoryMiddleware) RetrieveByID(ctx context.Context, id string) (users.User, error) {
	span := dbutil.CreateSpan(ctx, urm.tracer, retrieveUserByID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.RetrieveByID(ctx, id)
}

func (urm userRepositoryMiddleware) UpdatePassword(ctx context.Context, email, password string) error {
	span := dbutil.CreateSpan(ctx, urm.tracer, updateUserPassword)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.UpdatePassword(ctx, email, password)
}

func (urm userRepositoryMiddleware) RetrieveByIDs(ctx context.Context, ids []string, pm users.PageMetadata) (users.UserPage, error) {
	span := dbutil.CreateSpan(ctx, urm.tracer, retrieveUsersByIDs)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.RetrieveByIDs(ctx, ids, pm)
}

func (urm userRepositoryMiddleware) BackupAll(ctx context.Context) ([]users.User, error) {
	span := dbutil.CreateSpan(ctx, urm.tracer, backupAllUsers)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.BackupAll(ctx)
}

func (urm userRepositoryMiddleware) ChangeStatus(ctx context.Context, id, status string) error {
	span := dbutil.CreateSpan(ctx, urm.tracer, changeUserStatus)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.ChangeStatus(ctx, id, status)
}
