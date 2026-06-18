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
	"id":          "id",
	"occurred_at": "occurred_at",
	"operation":   "operation",
	"actor_email": "actor_email",
	"org_id":      "org_id",
	"group_id":    "group_id",
}

type Event struct {
	ID         string
	OccurredAt time.Time
	Operation  string
	Actor      domain.Identity
	OrgID      string
	GroupID    string
	ActionData map[string]any
}

type EventsPage struct {
	Total  uint64  `json:"total"`
	Events []Event `json:"events"`
}

type PageMetadata struct {
	Total      uint64         `json:"total,omitempty"`
	Offset     uint64         `json:"offset,omitempty"`
	Limit      uint64         `json:"limit,omitempty"`
	Order      string         `json:"order,omitempty"`
	Dir        string         `json:"dir,omitempty"`
	Email      string         `json:"email,omitempty"`
	Operation  string         `json:"operation,omitempty"`
	ActionData map[string]any `json:"action_data,omitempty"`
	From       time.Time      `json:"from,omitzero"`
	To         time.Time      `json:"to,omitzero"`
}

func (pm PageMetadata) Validate(maxLimitSize int) error {
	common := apiutil.PageMetadata{Offset: pm.Offset, Limit: pm.Limit, Order: pm.Order, Dir: pm.Dir}

	return common.Validate(maxLimitSize, AllowedOrders)
}

type EventRepository interface {
	// SaveEvent persists a single event to the database.
	SaveEvent(ctx context.Context, e Event) error

	// RetrieveEvents retrieves events from the database
	RetrieveEvents(ctx context.Context, pm PageMetadata) (EventsPage, error)

	// RetrieveEventsByOrg retrieves events belonging to a specific organization from the database
	RetrieveEventsByOrg(ctx context.Context, orgID string, pm PageMetadata) (EventsPage, error)

	// RetrieveEventsByGroup retrieves events belonging to a specific group from the database
	RetrieveEventsByGroup(ctx context.Context, groupID string, pm PageMetadata) (EventsPage, error)
}

type Service interface {
	// RecordEvent records a single event.
	RecordEvent(ctx context.Context, e events.Event) error

	// ListEvents retrieves a list of audit events across all organizations.
	// The user authenticated by `token` must be a platform (root) admin.
	ListEvents(ctx context.Context, token string, pm PageMetadata) (EventsPage, error)

	// ListEventsByOrg retrieves a list of audit events occurred in a specific organization denoted by its ID.
	// The user authenticated by `token` must possess "admin" or higher privileges within the target organization.
	ListEventsByOrg(ctx context.Context, token string, orgID string, pm PageMetadata) (EventsPage, error)

	// ListEventsByGroup retrieves a list of audit events occurred in a specific group denoted by its ID.
	// The user authenticated by `token` must possess "admin" or higher privileges within the target group.
	ListEventsByGroup(ctx context.Context, token string, groupID string, pm PageMetadata) (EventsPage, error)
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
		ID:         id,
		OccurredAt: occurredAt,
		Operation:  e.Action.Operation(),
		Actor:      e.JWTUserIdentity,
		OrgID:      e.OrgID,
		GroupID:    e.GroupID,
		ActionData: e.Action.Encode(),
	})
}

func (svc auditService) ListEvents(ctx context.Context, token string, pm PageMetadata) (EventsPage, error) {
	// Ensure that the authenticated user is the platform (root) admin
	if err := svc.auth.Authorize(ctx, domain.AuthzReq{
		Token:   token,
		Subject: domain.RootSub,
	}); err != nil {
		return EventsPage{}, err
	}

	return svc.events.RetrieveEvents(ctx, pm)
}

func (svc auditService) ListEventsByOrg(ctx context.Context, token string, orgID string, pm PageMetadata) (EventsPage, error) {
	// Ensure that the authenticated user has admin (or higher) privileges within the target Organization
	if err := svc.auth.Authorize(ctx, domain.AuthzReq{
		Token:   token,
		Object:  orgID,
		Subject: domain.OrgSub,
		Action:  domain.OrgAdmin,
	}); err != nil {
		return EventsPage{}, err
	}

	return svc.events.RetrieveEventsByOrg(ctx, orgID, pm)
}

func (svc auditService) ListEventsByGroup(ctx context.Context, token string, groupID string, pm PageMetadata) (EventsPage, error) {
	// Ensure that the authenticated user has admin (or higher) privileges within the target Group
	if err := svc.things.CanUserAccessGroup(ctx, domain.UserAccessReq{
		Token:  token,
		ID:     groupID,
		Action: domain.GroupAdmin,
	}); err != nil {
		return EventsPage{}, err
	}

	return svc.events.RetrieveEventsByGroup(ctx, groupID, pm)
}
