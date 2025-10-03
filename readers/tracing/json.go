// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/opentracing/opentracing-go"
)

const (
	retrieveJSONMessages = "retrieve_json_messages"
	backupJSONMessaages  = "backup_json_messages"
	restoreJSONMessages  = "restore_json_messages"
	removeJSONMessages   = "remove_json_messages"
)

var _ readers.JSONMessageRepository = (*jsonRepositoryMiddleware)(nil)

type jsonRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   readers.JSONMessageRepository
}

func JSONRepositoryMiddleware(tracer opentracing.Tracer, repo readers.JSONMessageRepository) readers.JSONMessageRepository {
	return jsonRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (jrm jsonRepositoryMiddleware) Retrieve(ctx context.Context, rpm readers.JSONPageMetadata) (readers.JSONMessagesPage, error) {
	span := createSpan(ctx, jrm.tracer, retrieveJSONMessages)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return jrm.repo.Retrieve(ctx, rpm)
}

func (jrm jsonRepositoryMiddleware) Backup(ctx context.Context, rpm readers.JSONPageMetadata) (readers.JSONMessagesPage, error) {
	span := createSpan(ctx, jrm.tracer, backupJSONMessaages)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return jrm.repo.Backup(ctx, rpm)
}

func (jrm jsonRepositoryMiddleware) Restore(ctx context.Context, messages ...readers.Message) error {
	span := createSpan(ctx, jrm.tracer, restoreJSONMessages)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return jrm.repo.Restore(ctx, messages...)
}

func (jrm jsonRepositoryMiddleware) Remove(ctx context.Context, rpm readers.JSONPageMetadata) error {
	span := createSpan(ctx, jrm.tracer, removeJSONMessages)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return jrm.repo.Remove(ctx, rpm)
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
