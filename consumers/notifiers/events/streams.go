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
	case events.GroupRemove:
		re := decodeRemoveGroupEvent(msg)
		if err := es.svc.RemoveNotifiersByGroup(ctx, re.id); err != nil {
			return err
		}
	}

	return nil
}

func decodeRemoveGroupEvent(event map[string]interface{}) removeGroupEvent {
	return removeGroupEvent{
		id: events.ReadField(event, "id", ""),
	}
}
