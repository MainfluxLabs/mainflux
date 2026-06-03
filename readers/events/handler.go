// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/events"
	"github.com/MainfluxLabs/mainflux/readers"
)

type eventHandler struct {
	svc readers.Service
}

// NewEventHandler returns new event store handler.
func NewEventHandler(svc readers.Service) events.EventHandler {
	return &eventHandler{svc: svc}
}

func (h *eventHandler) Handle(ctx context.Context, event events.Event) error {
	switch e := event.Action.(type) {
	case events.ThingRemoved:
		return h.svc.RemoveMessagesByThing(ctx, e.ID)
	case events.GroupRemoved:
		for _, thingID := range e.ThingIDs {
			if err := h.svc.RemoveMessagesByThing(ctx, thingID); err != nil {
				return err
			}
		}
	}
	return nil
}
