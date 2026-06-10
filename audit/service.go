// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package audit

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/events"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
)

var AllowedOrders = map[string]string{
	"id":               "id",
	"occurred_at":      "occurred_at",
	"operation":        "operation",
	"actor_user_email": "actor_user_email",
	"org_id":           "org_id",
	"group_id":         "group_id",
}

type Event struct {
	ID             string
	OccurredAt     time.Time
	Operation      string
	ActorUserID    string
	ActorUserEmail string
	OrgID          string
	GroupID        string
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
	OrgID     string         `json:"org_id,omitempty"`
	GroupID   string         `json:"group_id,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
}

func (pm PageMetadata) Validate(maxLimitSize int) error {
	common := apiutil.PageMetadata{Offset: pm.Offset, Limit: pm.Limit, Order: pm.Order, Dir: pm.Dir}

	return common.Validate(maxLimitSize, AllowedOrders)
}

type EventRepository interface {
	SaveEvent(ctx context.Context, e Event) error
	RetrieveEvents(ctx context.Context, pm PageMetadata) (EventsPage, error)
}

type Service interface {
	// RecordEvent persists a single event to the database.
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
		return err
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
		OrgID:          e.OrgID,
		GroupID:        e.GroupID,
		Data:           e.Action.Encode(),
	})
}
