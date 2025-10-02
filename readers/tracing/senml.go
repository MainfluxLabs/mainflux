// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/opentracing/opentracing-go"
)

const (
	senmlRetrieveMessages = "senml_retrieve_messages"
	senmlBackupMessaages  = "senml_backup_messages"
	senmlRestoreMessages  = "senml_restore_messages"
	senmlRemoveMessages   = "senml_remove_messages"
)

var _ readers.SenMLMessageRepository = (*senmlMessageRepositoryMiddleware)(nil)

type senmlMessageRepositoryMiddleware struct {
	tracer          opentracing.Tracer
	senmlRepository readers.SenMLMessageRepository
}

func SenMLMessageRepositoryMiddleware(tracer opentracing.Tracer, senmlRepository readers.SenMLMessageRepository) readers.SenMLMessageRepository {
	return senmlMessageRepositoryMiddleware{
		tracer:          tracer,
		senmlRepository: senmlRepository,
	}
}

func (srm senmlMessageRepositoryMiddleware) Retrieve(ctx context.Context, rpm readers.SenMLPageMetadata) (readers.SenMLMessagesPage, error) {
	span := createSpan(ctx, srm.tracer, senmlRetrieveMessages)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return srm.senmlRepository.Retrieve(ctx, rpm)
}

func (srm senmlMessageRepositoryMiddleware) Backup(ctx context.Context, rpm readers.SenMLPageMetadata) (readers.SenMLMessagesPage, error) {
	span := createSpan(ctx, srm.tracer, senmlBackupMessaages)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return srm.senmlRepository.Backup(ctx, rpm)
}

func (srm senmlMessageRepositoryMiddleware) Restore(ctx context.Context, messages ...readers.Message) error {
	span := createSpan(ctx, srm.tracer, senmlRestoreMessages)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return srm.senmlRepository.Restore(ctx, messages...)
}

func (srm senmlMessageRepositoryMiddleware) Remove(ctx context.Context, rpm readers.SenMLPageMetadata) error {
	span := createSpan(ctx, srm.tracer, senmlRemoveMessages)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return srm.senmlRepository.Remove(ctx, rpm)
}
