// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"github.com/MainfluxLabs/mainflux/audit"
	opentracing "github.com/opentracing/opentracing-go"
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
