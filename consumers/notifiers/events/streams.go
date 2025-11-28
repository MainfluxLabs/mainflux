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
	//TODO: Use const
	case "group.remove":
		re := decodeRemoveGroupEvent(msg)
		if err := es.svc.RemoveNotifiersByGroup(ctx, re.id); err != nil {
			return err
		}
	}

	return nil
}

func decodeRemoveGroupEvent(event map[string]interface{}) removeGroupEvent {
	val, ok := event["id"].(string)
	if !ok {
		return removeGroupEvent{id: ""}
	}

	return removeGroupEvent{
		id: val,
	}
}
