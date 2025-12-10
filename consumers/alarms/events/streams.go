package events

import (
	"context"

	"github.com/MainfluxLabs/mainflux/consumers/alarms"
	"github.com/MainfluxLabs/mainflux/pkg/events"
)

type eventHandler struct {
	svc alarms.Service
}

// NewEventHandler returns new event store handler.
func NewEventHandler(svc alarms.Service) events.EventHandler {
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
		if err := es.svc.RemoveAlarmsByThing(ctx, re.id); err != nil {
			return err
		}
	case events.GroupRemove:
		re := decodeRemoveEvent(msg)
		if err := es.svc.RemoveAlarmsByGroup(ctx, re.id); err != nil {
			return err
		}
	}

	return nil
}

func decodeRemoveEvent(event map[string]any) removeEvent {
	return removeEvent{
		id: events.ReadField(event, "id", ""),
	}
}
