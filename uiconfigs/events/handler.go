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
	switch e := event.Action.(type) {
	case events.ThingRemoved:
		if err := h.svc.RemoveThingConfig(ctx, e.ID); err != nil {
			return err
		}
		return h.svc.RemoveThingsFromConfigs(ctx, event.OrgID, []string{e.ID})
	case events.OrgRemoved:
		return h.svc.RemoveOrgConfig(ctx, e.ID)
	case events.GroupRemoved:
		if err := h.svc.RemoveThingConfigByGroup(ctx, e.ID); err != nil {
			return err
		}
		return h.svc.RemoveGroupFromConfigs(ctx, event.OrgID, e.ID, e.ThingIDs)
	}
	return nil
}
