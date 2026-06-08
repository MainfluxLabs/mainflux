// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/events"
	"github.com/MainfluxLabs/mainflux/webhooks"
)

type eventHandler struct {
	svc webhooks.Service
}

// NewEventHandler returns new event store handler.
func NewEventHandler(svc webhooks.Service) events.EventHandler {
	return &eventHandler{svc: svc}
}

func (h *eventHandler) Handle(ctx context.Context, event events.Event) error {
	switch e := event.(type) {
	case events.ThingRemoved:
		return h.svc.RemoveWebhooksByThing(ctx, e.ID)
	case events.GroupRemoved:
		return h.svc.RemoveWebhooksByGroup(ctx, e.ID)
	}
	return nil
}
