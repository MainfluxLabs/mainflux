// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/events"
	"github.com/MainfluxLabs/mainflux/shadows"
)

type eventHandler struct {
	svc shadows.Service
}

// NewEventHandler returns new event store handler.
func NewEventHandler(svc shadows.Service) events.EventHandler {
	return &eventHandler{svc: svc}
}

func (h *eventHandler) Handle(ctx context.Context, event events.Event) error {
	switch e := event.Action.(type) {
	case events.ThingRemoved:
		return h.svc.RemoveByThing(ctx, e.ID)
	}
	return nil
}
