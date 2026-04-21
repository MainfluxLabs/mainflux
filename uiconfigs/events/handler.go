// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/events"
	"github.com/MainfluxLabs/mainflux/uiconfigs"
)

type eventHandler struct {
	svc uiconfigs.Service
}

// NewEventHandler returns new event store handler.
func NewEventHandler(svc uiconfigs.Service) events.EventHandler {
	return &eventHandler{svc: svc}
}

func (h *eventHandler) Handle(ctx context.Context, event events.Event) error {
	switch e := event.(type) {
	case events.ThingRemoved:
		return h.svc.RemoveThingConfig(ctx, e.ID)
	case events.OrgRemoved:
		return h.svc.RemoveOrgConfig(ctx, e.ID)
	case events.GroupRemoved:
		return h.svc.RemoveThingConfigByGroup(ctx, e.ID)
	}
	return nil
}
