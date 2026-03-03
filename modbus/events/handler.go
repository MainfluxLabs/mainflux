package events

import (
	"context"

	"github.com/MainfluxLabs/mainflux/modbus"
	"github.com/MainfluxLabs/mainflux/pkg/events"
)

type eventHandler struct {
	svc modbus.Service
}

// NewEventHandler returns new event store handler.
func NewEventHandler(svc modbus.Service) events.EventHandler {
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
		if err := es.svc.RemoveClientsByThing(ctx, re.id); err != nil {
			return err
		}
	case events.ProfileUpdate:
		upe, err := decodeUpdateProfileEvent(msg)
		if err != nil {
			return err
		}
		if err := es.svc.RescheduleTasks(ctx, upe.id, upe.config); err != nil {
			return err
		}
	case events.GroupRemove:
		re := decodeRemoveEvent(msg)
		if err := es.svc.RemoveClientsByGroup(ctx, re.id); err != nil {
			return err
		}
	}

	return nil
}
