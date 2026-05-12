package events

import (
	"context"
	"strings"

	"github.com/MainfluxLabs/mainflux/pkg/events"
	"github.com/MainfluxLabs/mainflux/readers"
)

type eventHandler struct {
	svc readers.Service
}

// NewEventHandler returns new event store handler.
func NewEventHandler(svc readers.Service) events.EventHandler {
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
		re := decodeRemoveThingEvent(msg)
		if err := es.svc.RemoveMessagesByThing(ctx, re.id); err != nil {
			return err
		}
	case events.GroupRemove:
		re := decodeRemoveGroupEvent(msg)
		for _, thingID := range re.thingIDs {
			if err := es.svc.RemoveMessagesByThing(ctx, thingID); err != nil {
				return err
			}
		}
	}

	return nil
}

func decodeRemoveThingEvent(event map[string]any) removeThingEvent {
	return removeThingEvent{
		id: events.ReadField(event, "id", ""),
	}
}

func decodeRemoveGroupEvent(event map[string]any) removeGroupEvent {
	raw := events.ReadField(event, "thing_ids", "")
	if raw == "" {
		return removeGroupEvent{}
	}
	return removeGroupEvent{
		thingIDs: strings.Split(raw, ","),
	}
}
