// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/MainfluxLabs/mainflux/mqtt"
	"github.com/MainfluxLabs/mainflux/pkg/events"
)

type eventHandler struct {
	svc mqtt.Service
}

// NewEventHandler returns new event store handler.
func NewEventHandler(svc mqtt.Service) events.EventHandler {
	return &eventHandler{svc: svc}
}

func (h *eventHandler) Handle(ctx context.Context, event events.Event) error {
	switch e := event.(type) {
	case events.ThingRemoved:
		return h.svc.RemoveSubscriptionsByThing(ctx, e.ID)
	case events.GroupRemoved:
		return h.svc.RemoveSubscriptionsByGroup(ctx, e.ID)
	}
	return nil
}
