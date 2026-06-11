// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/audit"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	opentracing "github.com/opentracing/opentracing-go"
)

const (
	saveEvent           = "save_event"
	retrieveEventsByOrg = "retrieve_events_by_org"
)

var _ audit.EventRepository = (*eventRepositoryMiddleware)(nil)

type eventRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   audit.EventRepository
}

func NewEventRepositoryMiddleware(tracer opentracing.Tracer, repo audit.EventRepository) audit.EventRepository {
	return &eventRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (m *eventRepositoryMiddleware) SaveEvent(ctx context.Context, e audit.Event) error {
	span := dbutil.CreateSpan(ctx, m.tracer, saveEvent)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)
	return m.repo.SaveEvent(ctx, e)
}

func (m *eventRepositoryMiddleware) RetrieveEventsByOrg(ctx context.Context, orgID string, pm audit.PageMetadata) (audit.EventsPage, error) {
	span := dbutil.CreateSpan(ctx, m.tracer, retrieveEventsByOrg)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)
	return m.repo.RetrieveEventsByOrg(ctx, orgID, pm)
}
