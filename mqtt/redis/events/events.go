package events

import (
	"github.com/MainfluxLabs/mainflux/pkg/events"
)

type removeEvent struct {
	id string
}

func decodeRemoveEvent(event map[string]any) removeEvent {
	return removeEvent{
		id: events.ReadField(event, "id", ""),
	}
}
