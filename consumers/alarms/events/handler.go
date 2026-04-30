// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/MainfluxLabs/mainflux/consumers/alarms"
	"github.com/MainfluxLabs/mainflux/pkg/events"
)

type eventHandler struct {
	svc alarms.Service
}

// NewEventHandler returns new event store handler.
func NewEventHandler(svc alarms.Service) events.EventHandler {
	return &eventHandler{svc: svc}
}

func (h *eventHandler) Handle(ctx context.Context, event events.Event) error {
	switch e := event.(type) {
	case events.ThingRemoved:
		return h.svc.RemoveAlarmsByThing(ctx, e.ID)
	case events.GroupRemoved:
		return h.svc.RemoveAlarmsByGroup(ctx, e.ID)
	}
	return nil
}
