// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package tracing contains middlewares that will add spans
// to existing traces.
package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/auth"
	opentracing "github.com/opentracing/opentracing-go"
)

const (
	saveKey     = "save_key"
	retrieveKey = "retrieve_key_by_id"
	removeKey   = "remove_key"
)

var _ auth.KeyRepository = (*keyRepositoryMiddleware)(nil)

// keyRepositoryMiddleware tracks request and their latency, and adds spans
// to context.
type keyRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   auth.KeyRepository
}

// New tracks request and their latency, and adds spans
// to context.
func New(repo auth.KeyRepository, tracer opentracing.Tracer) auth.KeyRepository {
	return keyRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (krm keyRepositoryMiddleware) Save(ctx context.Context, key auth.Key) (string, error) {
	span := createSpan(ctx, krm.tracer, saveKey)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return krm.repo.Save(ctx, key)
}

func (krm keyRepositoryMiddleware) Retrieve(ctx context.Context, owner, id string) (auth.Key, error) {
	span := createSpan(ctx, krm.tracer, retrieveKey)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return krm.repo.Retrieve(ctx, owner, id)
}

func (krm keyRepositoryMiddleware) Remove(ctx context.Context, owner, id string) error {
	span := createSpan(ctx, krm.tracer, removeKey)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return krm.repo.Remove(ctx, owner, id)
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
