package events

import (
	"context"

	"github.com/MainfluxLabs/mainflux/mqtt"
	"github.com/MainfluxLabs/mainflux/pkg/events"
)

type eventHandler struct {
	svc mqtt.Service
}

// NewEventHandler returns new event store handler.
func NewEventHandler(svc mqtt.Service) events.EventHandler {
	return &eventHandler{
		svc: svc,
	}
}

func (es eventHandler) Handle(ctx context.Context, event events.Event) error {
	msg, err := event.Encode()
	if err != nil {
		return err
	}

	switch msg["operation"] {
	case events.ThingRemove:
		re := decodeRemoveEvent(msg)
		if err := es.svc.RemoveSubscriptionsByThing(ctx, re.id); err != nil {
			return err
		}
	case events.GroupRemove:
		re := decodeRemoveEvent(msg)
		if err := es.svc.RemoveSubscriptionsByGroup(ctx, re.id); err != nil {
			return err
		}
	}

	return nil
}
