package events

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/events"
	"github.com/MainfluxLabs/mainflux/webhooks"
)

type eventHandler struct {
	svc webhooks.Service
}

// NewEventHandler returns new event store handler.
func NewEventHandler(svc webhooks.Service) events.EventHandler {
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
		if err := es.svc.RemoveWebhooksByThing(ctx, re.id); err != nil {
			return err
		}
	case events.GroupRemove:
		re := decodeRemoveEvent(msg)
		if err := es.svc.RemoveWebhooksByGroup(ctx, re.id); err != nil {
			return err
		}
	}

	return nil
}
