// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/events"
	"github.com/MainfluxLabs/mainflux/pyscripts"
)

type eventHandler struct {
	svc pyscripts.Service
}

// NewEventHandler returns new event store handler.
func NewEventHandler(svc pyscripts.Service) events.EventHandler {
	return &eventHandler{svc: svc}
}

func (h *eventHandler) Handle(ctx context.Context, event events.Event) error {
	switch e := event.Action.(type) {
	case events.GroupRemoved:
		return h.svc.RemoveScriptsByGroup(ctx, e.ID)
	}

	return nil
}
