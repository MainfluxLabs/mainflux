// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/MainfluxLabs/mainflux/downlinks"
	"github.com/MainfluxLabs/mainflux/pkg/events"
)

type eventHandler struct {
	svc downlinks.Service
}

// NewEventHandler returns new event store handler.
func NewEventHandler(svc downlinks.Service) events.EventHandler {
	return &eventHandler{svc: svc}
}

func (h *eventHandler) Handle(ctx context.Context, event events.Event) error {
	switch e := event.(type) {
	case events.ThingRemoved:
		return h.svc.RemoveDownlinksByThing(ctx, e.ID)
	case events.ProfileUpdated:
		return h.svc.RescheduleTasks(ctx, e.ID, e.Config)
	case events.GroupRemoved:
		return h.svc.RemoveDownlinksByGroup(ctx, e.ID)
	}
	return nil
}
