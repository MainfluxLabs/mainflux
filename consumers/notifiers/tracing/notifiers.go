// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/consumers/notifiers"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/opentracing/opentracing-go"
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

func (n notifierRepositoryMiddleware) Save(ctx context.Context, nfs ...things.Notifier) ([]things.Notifier, error) {
	span := createSpan(ctx, n.tracer, "save_notifiers")
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return n.repo.Save(ctx, nfs...)
}

func (n notifierRepositoryMiddleware) RetrieveByGroupID(ctx context.Context, groupID string) ([]things.Notifier, error) {
	span := createSpan(ctx, n.tracer, "retrieve_by_group_id")
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return n.repo.RetrieveByGroupID(ctx, groupID)
}

func (n notifierRepositoryMiddleware) RetrieveByID(ctx context.Context, id string) (things.Notifier, error) {
	span := createSpan(ctx, n.tracer, "retrieve_notifier_by_id")
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return n.repo.RetrieveByID(ctx, id)
}

func (n notifierRepositoryMiddleware) Update(ctx context.Context, ntf things.Notifier) error {
	span := createSpan(ctx, n.tracer, "update_notifier")
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return n.repo.Update(ctx, ntf)
}

func (n notifierRepositoryMiddleware) Remove(ctx context.Context, groupID string, ids ...string) error {
	span := createSpan(ctx, n.tracer, "remove_notifiers")
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return n.repo.Remove(ctx, groupID, ids...)
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
