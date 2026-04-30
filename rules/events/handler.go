// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/events"
	"github.com/MainfluxLabs/mainflux/rules"
)

type eventHandler struct {
	svc rules.Service
}

// NewEventHandler returns new event store handler.
func NewEventHandler(svc rules.Service) events.EventHandler {
	return &eventHandler{svc: svc}
}

func (h *eventHandler) Handle(ctx context.Context, event events.Event) error {
	switch e := event.(type) {
	case events.ThingRemoved:
		if err := h.svc.UnassignRulesByThing(ctx, e.ID); err != nil {
			return err
		}
		if err := h.svc.UnassignScriptsFromThing(ctx, e.ID); err != nil {
			return err
		}
	case events.GroupRemoved:
		if err := h.svc.RemoveRulesByGroup(ctx, e.ID); err != nil {
			return err
		}
		if err := h.svc.RemoveScriptsByGroup(ctx, e.ID); err != nil {
			return err
		}
	}
	return nil
}
