// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/consumers/notifiers"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/opentracing/opentracing-go"
)

const (
	saveNotifiers            = "save_notifiers"
	retrieveNotifiersByGroup = "retrieve_notifiers_by_group"
	retrieveNotifierByID     = "retrieve_notifier_by_id"
	updateNotifier           = "update_notifier"
	removeNotifiers          = "remove_notifiers"
	removeNotifiersByGroup   = "remove_notifiers_by_group"
)

var (
	_ notifiers.NotifierRepository = (*notifierRepositoryMiddleware)(nil)
)

type notifierRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   notifiers.NotifierRepository
}

// NotifierRepositoryMiddleware tracks request and their latency, and adds spans
// to context.
func NotifierRepositoryMiddleware(tracer opentracing.Tracer, repo notifiers.NotifierRepository) notifiers.NotifierRepository {
	return notifierRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (n notifierRepositoryMiddleware) Save(ctx context.Context, nfs ...notifiers.Notifier) ([]notifiers.Notifier, error) {
	span := dbutil.CreateSpan(ctx, n.tracer, saveNotifiers)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return n.repo.Save(ctx, nfs...)
}

func (n notifierRepositoryMiddleware) RetrieveByGroup(ctx context.Context, groupID string, pm apiutil.PageMetadata) (notifiers.NotifiersPage, error) {
	span := dbutil.CreateSpan(ctx, n.tracer, retrieveNotifiersByGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return n.repo.RetrieveByGroup(ctx, groupID, pm)
}

func (n notifierRepositoryMiddleware) RetrieveByID(ctx context.Context, id string) (notifiers.Notifier, error) {
	span := dbutil.CreateSpan(ctx, n.tracer, retrieveNotifierByID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return n.repo.RetrieveByID(ctx, id)
}

func (n notifierRepositoryMiddleware) Update(ctx context.Context, ntf notifiers.Notifier) error {
	span := dbutil.CreateSpan(ctx, n.tracer, updateNotifier)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return n.repo.Update(ctx, ntf)
}

func (n notifierRepositoryMiddleware) Remove(ctx context.Context, ids ...string) error {
	span := dbutil.CreateSpan(ctx, n.tracer, removeNotifiers)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return n.repo.Remove(ctx, ids...)
}

func (n notifierRepositoryMiddleware) RemoveByGroup(ctx context.Context, groupID string) error {
	span := dbutil.CreateSpan(ctx, n.tracer, removeNotifiersByGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return n.repo.RemoveByGroup(ctx, groupID)
}
