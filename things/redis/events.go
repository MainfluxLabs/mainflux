package redis

import (
	"encoding/json"

	"github.com/MainfluxLabs/mainflux/pkg/events"
)

type event interface {
	Encode() map[string]any
}

var (
	_ event = (*createThingEvent)(nil)
	_ event = (*updateThingEvent)(nil)
	_ event = (*removeThingEvent)(nil)
	_ event = (*createProfileEvent)(nil)
	_ event = (*updateProfileEvent)(nil)
	_ event = (*removeProfileEvent)(nil)
)

type createThingEvent struct {
	id        string
	groupID   string
	profileID string
	name      string
	metadata  map[string]any
}

func (cte createThingEvent) Encode() map[string]any {
	val := map[string]any{
		"id":         cte.id,
		"group_id":   cte.groupID,
		"profile_id": cte.profileID,
		"operation":  events.ThingCreate,
	}

	if cte.name != "" {
		val["name"] = cte.name
	}

	if cte.metadata != nil {
		metadata, err := json.Marshal(cte.metadata)
		if err != nil {
			return val
		}

		val["metadata"] = string(metadata)
	}

	return val
}

type updateThingEvent struct {
	id        string
	profileID string
	name      string
	metadata  map[string]any
}

func (ute updateThingEvent) Encode() map[string]any {
	val := map[string]any{
		"id":         ute.id,
		"profile_id": ute.profileID,
		"operation":  events.ThingUpdate,
	}

	if ute.name != "" {
		val["name"] = ute.name
	}

	if ute.metadata != nil {
		metadata, err := json.Marshal(ute.metadata)
		if err != nil {
			return val
		}

		val["metadata"] = string(metadata)
	}

	return val
}

type updateThingGroupAndProfileEvent struct {
	id        string
	profileID string
	groupID   string
}

func (pte updateThingGroupAndProfileEvent) Encode() map[string]any {
	val := map[string]any{
		"id":         pte.id,
		"profile_id": pte.profileID,
		"operation":  events.ThingUpdateGroupAndProfile,
	}

	if pte.groupID != "" {
		val["groupID"] = pte.groupID
	}

	return val
}

type removeThingEvent struct {
	id string
}

func (rte removeThingEvent) Encode() map[string]any {
	return map[string]any{
		"id":        rte.id,
		"operation": events.ThingRemove,
	}
}

type createProfileEvent struct {
	id       string
	groupID  string
	name     string
	metadata map[string]any
}

func (cpe createProfileEvent) Encode() map[string]any {
	val := map[string]any{
		"id":        cpe.id,
		"group_id":  cpe.groupID,
		"operation": events.ProfileCreate,
	}

	if cpe.name != "" {
		val["name"] = cpe.name
	}

	if cpe.metadata != nil {
		metadata, err := json.Marshal(cpe.metadata)
		if err != nil {
			return val
		}

		val["metadata"] = string(metadata)
	}

	return val
}

type updateProfileEvent struct {
	id       string
	name     string
	config   map[string]any
	metadata map[string]any
}

func (upe updateProfileEvent) Encode() map[string]any {
	val := map[string]any{
		"id":        upe.id,
		"operation": events.ProfileUpdate,
	}

	if upe.name != "" {
		val["name"] = upe.name
	}

	if upe.config != nil {
		config, err := json.Marshal(upe.config)
		if err != nil {
			return val
		}

		val["config"] = string(config)
	}

	if upe.metadata != nil {
		metadata, err := json.Marshal(upe.metadata)
		if err != nil {
			return val
		}

		val["metadata"] = string(metadata)
	}

	return val
}

type removeProfileEvent struct {
	id string
}

func (rpe removeProfileEvent) Encode() map[string]any {
	return map[string]any{
		"id":        rpe.id,
		"operation": events.ProfileRemove,
	}
}

type removeGroupEvent struct {
	id string
}

func (rge removeGroupEvent) Encode() map[string]any {
	return map[string]any{
		"id":        rge.id,
		"operation": events.GroupRemove,
	}
}

type removeOrgEvent struct {
	id string
}

func decodeRemoveOrgEvent(event map[string]any) removeOrgEvent {
	return removeOrgEvent{
		id: events.ReadField(event, "id", ""),
	}
}
