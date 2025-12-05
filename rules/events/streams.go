package events

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/events"
	"github.com/MainfluxLabs/mainflux/rules"
)

type eventHandler struct {
	svc rules.Service
}

// NewEventHandler returns new event store handler.
func NewEventHandler(svc rules.Service) events.EventHandler {
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
		if err := es.svc.UnassignRulesByThing(ctx, re.id); err != nil {
			return err
		}
	case events.GroupRemove:
		re := decodeRemoveEvent(msg)
		if err := es.svc.RemoveRulesByGroup(ctx, re.id); err != nil {
			return err
		}
	}

	return nil
}

func decodeRemoveEvent(event map[string]interface{}) removeEvent {
	return removeEvent{
		id: events.ReadField(event, "id", ""),
	}
}
