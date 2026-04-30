// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/MainfluxLabs/mainflux/consumers/notifiers"
	"github.com/MainfluxLabs/mainflux/pkg/events"
)

type eventHandler struct {
	svc notifiers.Service
}

// NewEventHandler returns new event store handler.
func NewEventHandler(svc notifiers.Service) events.EventHandler {
	return &eventHandler{svc: svc}
}

func (h *eventHandler) Handle(ctx context.Context, event events.Event) error {
	switch e := event.(type) {
	case events.GroupRemoved:
		return h.svc.RemoveNotifiersByGroup(ctx, e.ID)
	}
	return nil
}
