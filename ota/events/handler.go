// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/MainfluxLabs/mainflux/ota"
	"github.com/MainfluxLabs/mainflux/pkg/events"
)

type eventHandler struct {
	svc ota.Service
}

// NewEventHandler returns new event store handler.
func NewEventHandler(svc ota.Service) events.EventHandler {
	return &eventHandler{svc: svc}
}

func (h *eventHandler) Handle(ctx context.Context, event events.Event) error {
	switch e := event.Action.(type) {
	case events.ThingRemoved:
		return h.svc.RemoveThingUpdatesByThing(ctx, e.ID)
	case events.GroupRemoved:
		return h.svc.RemoveUpdatesByGroup(ctx, e.ID)
	}
	return nil
}
