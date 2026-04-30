// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/MainfluxLabs/mainflux/modbus"
	"github.com/MainfluxLabs/mainflux/pkg/events"
)

type eventHandler struct {
	svc modbus.Service
}

// NewEventHandler returns new event store handler.
func NewEventHandler(svc modbus.Service) events.EventHandler {
	return &eventHandler{svc: svc}
}

func (h *eventHandler) Handle(ctx context.Context, event events.Event) error {
	switch e := event.(type) {
	case events.ThingRemoved:
		return h.svc.RemoveClientsByThing(ctx, e.ID)
	case events.ProfileUpdated:
		return h.svc.RescheduleTasks(ctx, e.ID, e.Config)
	case events.GroupRemoved:
		return h.svc.RemoveClientsByGroup(ctx, e.ID)
	}
	return nil
}
