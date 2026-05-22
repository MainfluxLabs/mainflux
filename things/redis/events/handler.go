// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/events"
	"github.com/MainfluxLabs/mainflux/things"
)

type eventHandler struct {
	svc things.Service
}

// NewEventHandler returns new event store handler.
func NewEventHandler(svc things.Service) events.EventHandler {
	return &eventHandler{svc: svc}
}

func (h *eventHandler) Handle(ctx context.Context, event events.Event) error {
	switch e := event.(type) {
	case events.OrgRemoved:
		if _, err := h.svc.RemoveGroupsByOrg(ctx, e.ID); err != nil {
			return err
		}
	}
	return nil
}
