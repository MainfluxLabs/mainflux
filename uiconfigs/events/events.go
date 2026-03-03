package events

import (
	"github.com/MainfluxLabs/mainflux/pkg/events"
)

type event struct {
	id string
}

func decodeEvent(e map[string]any) event {
	return event{
		id: events.ReadField(e, "id", ""),
	}
}
