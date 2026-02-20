package events

import (
	"encoding/json"

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

type updateProfileEvent struct {
	config map[string]any
	id     string
}

func decodeUpdateProfileEvent(event map[string]any) (updateProfileEvent, error) {
	var config map[string]any
	if raw, ok := event["config"].(string); ok && raw != "" {
		if err := json.Unmarshal([]byte(raw), &config); err != nil {
			return updateProfileEvent{}, err
		}
	}

	return updateProfileEvent{
		id:     events.ReadField(event, "id", ""),
		config: config,
	}, nil
}
