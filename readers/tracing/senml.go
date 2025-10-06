// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/opentracing/opentracing-go"
)

const (
	retrieveSenMLMessages = "retrieve_senml_messages"
	backupSenMLMessages   = "backup_senml_messages"
	restoreSenMLMessages  = "restore_senml_messages"
	removeSenMLMessages   = "remove_senml_messages"
)

var _ readers.SenMLMessageRepository = (*senmlRepositoryMiddleware)(nil)

type senmlRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   readers.SenMLMessageRepository
}

func SenMLRepositoryMiddleware(tracer opentracing.Tracer, repo readers.SenMLMessageRepository) readers.SenMLMessageRepository {
	return senmlRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (srm senmlRepositoryMiddleware) Retrieve(ctx context.Context, rpm readers.SenMLPageMetadata) (readers.SenMLMessagesPage, error) {
	span := dbutil.CreateSpan(ctx, srm.tracer, retrieveSenMLMessages)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return srm.repo.Retrieve(ctx, rpm)
}

func (srm senmlRepositoryMiddleware) Backup(ctx context.Context, rpm readers.SenMLPageMetadata) (readers.SenMLMessagesPage, error) {
	span := dbutil.CreateSpan(ctx, srm.tracer, backupSenMLMessages)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return srm.repo.Backup(ctx, rpm)
}

func (srm senmlRepositoryMiddleware) Restore(ctx context.Context, messages ...readers.Message) error {
	span := dbutil.CreateSpan(ctx, srm.tracer, restoreSenMLMessages)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return srm.repo.Restore(ctx, messages...)
}

func (srm senmlRepositoryMiddleware) Remove(ctx context.Context, rpm readers.SenMLPageMetadata) error {
	span := dbutil.CreateSpan(ctx, srm.tracer, removeSenMLMessages)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return srm.repo.Remove(ctx, rpm)
}
