// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/opentracing/opentracing-go"
)

const (
	jsonRetrieveMessages = "json_retrieve_messages"
	jsonBackupMessaages  = "json_backup_messages"
	jsonRestoreMessages  = "json_restore_messages"
	jsonRemoveMessages   = "json_remove_messages"
)

var _ readers.JSONMessageRepository = (*jsonMessageRepositoryMiddleware)(nil)

type jsonMessageRepositoryMiddleware struct {
	tracer         opentracing.Tracer
	jsonRepository readers.JSONMessageRepository
}

func JSONMessageRepositoryMiddleware(tracer opentracing.Tracer, jsonRepository readers.JSONMessageRepository) readers.JSONMessageRepository {
	return jsonMessageRepositoryMiddleware{
		tracer:         tracer,
		jsonRepository: jsonRepository,
	}
}

func (jrm jsonMessageRepositoryMiddleware) Retrieve(ctx context.Context, rpm readers.JSONPageMetadata) (readers.JSONMessagesPage, error) {
	span := createSpan(ctx, jrm.tracer, jsonRetrieveMessages)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return jrm.jsonRepository.Retrieve(ctx, rpm)
}

func (jrm jsonMessageRepositoryMiddleware) Backup(ctx context.Context, rpm readers.JSONPageMetadata) (readers.JSONMessagesPage, error) {
	span := createSpan(ctx, jrm.tracer, jsonBackupMessaages)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return jrm.jsonRepository.Backup(ctx, rpm)
}

func (jrm jsonMessageRepositoryMiddleware) Restore(ctx context.Context, messages ...readers.Message) error {
	span := createSpan(ctx, jrm.tracer, jsonRestoreMessages)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return jrm.jsonRepository.Restore(ctx, messages...)
}

func (jrm jsonMessageRepositoryMiddleware) Remove(ctx context.Context, rpm readers.JSONPageMetadata) error {
	span := createSpan(ctx, jrm.tracer, jsonRemoveMessages)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return jrm.jsonRepository.Remove(ctx, rpm)
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
