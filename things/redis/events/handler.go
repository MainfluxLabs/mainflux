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
	return &eventHandler{
		svc: svc,
	}
}

func (e eventHandler) Handle(ctx context.Context, event events.Event) error {
	msg, err := event.Encode()
	if err != nil {
		return err
	}

	switch msg["operation"] {
	case events.OrgRemove:
		re := decodeRemoveOrgEvent(msg)
		if _, err := e.svc.RemoveGroupsByOrg(ctx, re.id); err != nil {
			return err
		}
		return nil
	}

	return nil
}
