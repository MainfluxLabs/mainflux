// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package audit

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/events"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
)

var errRecordEvent = errors.New("failed to record event")

var AllowedOrders = map[string]string{
	"id":               "id",
	"occurred_at":      "occurred_at",
	"operation":        "operation",
	"actor_user_email": "actor_user_email",
}

type Event struct {
	ID             string
	OccurredAt     time.Time
	Operation      string
	ActorUserID    string
	ActorUserEmail string
	Data           map[string]any
}

type EventsPage struct {
	PageMetadata
	Events []Event
}

type PageMetadata struct {
	Total     uint64         `json:"total,omitempty"`
	Offset    uint64         `json:"offset,omitempty"`
	Limit     uint64         `json:"limit,omitempty"`
	Order     string         `json:"order,omitempty"`
	Dir       string         `json:"dir,omitempty"`
	Email     string         `json:"email,omitempty"`
	Operation string         `json:"operation,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
}

type EventRepository interface {
	SaveEvent(ctx context.Context, e Event) error
	RetrieveEvents(ctx context.Context, pm PageMetadata) (EventsPage, error)
}

type Service interface {
	RecordEvent(ctx context.Context, e events.Event) error
}

var _ Service = (*auditService)(nil)

type auditService struct {
	events EventRepository
	auth   domain.AuthClient
	things domain.ThingsClient
	idp    uuid.IDProvider
}

func New(events EventRepository, auth domain.AuthClient, things domain.ThingsClient, idp uuid.IDProvider) Service {
	return &auditService{
		events: events,
		auth:   auth,
		things: things,
		idp:    idp,
	}
}

func (s *auditService) RecordEvent(ctx context.Context, e events.Event) error {
	id, err := s.idp.ID()
	if err != nil {
		return errors.Wrap(errRecordEvent, err)
	}

	occurredAt := e.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	return s.events.SaveEvent(ctx, Event{
		ID:             id,
		OccurredAt:     occurredAt,
		Operation:      e.Action.Operation(),
		ActorUserID:    e.JWTUserIdentity.ID,
		ActorUserEmail: e.JWTUserIdentity.Email,
		Data:           e.Action.Encode(),
	})
}
