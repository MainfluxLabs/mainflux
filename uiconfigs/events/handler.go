package events

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/events"
	"github.com/MainfluxLabs/mainflux/uiconfigs"
)

type eventHandler struct {
	svc uiconfigs.Service
}

// NewEventHandler returns new event store handler.
func NewEventHandler(svc uiconfigs.Service) events.EventHandler {
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
		e := decodeEvent(msg)
		if err := es.svc.RemoveThingConfig(ctx, e.id); err != nil {
			return err
		}
	case events.OrgRemove:
		e := decodeEvent(msg)
		if err := es.svc.RemoveOrgConfig(ctx, e.id); err != nil {
			return err
		}
	case events.GroupRemove:
		e := decodeEvent(msg)
		if err := es.svc.RemoveThingConfigByGroup(ctx, e.id); err != nil {
			return err
		}
	}

	return nil
}
