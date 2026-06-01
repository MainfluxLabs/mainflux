// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/MainfluxLabs/mainflux/certs"
	"github.com/MainfluxLabs/mainflux/pkg/events"
)

type eventHandler struct {
	svc certs.Service
}

// NewEventHandler returns new event store handler.
func NewEventHandler(svc certs.Service) events.EventHandler {
	return &eventHandler{svc: svc}
}

func (h *eventHandler) Handle(ctx context.Context, event events.Event) error {
	switch e := event.(type) {
	case events.ThingRemoved:
		return h.svc.RemoveCertsByThing(ctx, e.ID)
	case events.GroupRemoved:
		for _, thingID := range e.ThingIDs {
			if err := h.svc.RemoveCertsByThing(ctx, thingID); err != nil {
				return err
			}
		}
	}
	return nil
}
