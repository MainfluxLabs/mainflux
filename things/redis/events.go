package redis

import "encoding/json"

const (
	thingPrefix                = "thing."
	thingCreate                = thingPrefix + "create"
	thingUpdate                = thingPrefix + "update"
	thingUpdateGroupAndProfile = thingPrefix + "update_group_and_profile"
	thingRemove                = thingPrefix + "remove"

	profilePrefix = "profile."
	profileCreate = profilePrefix + "create"
	profileUpdate = profilePrefix + "update"
	profileRemove = profilePrefix + "remove"

	groupPrefix = "group."
	groupRemove = groupPrefix + "remove"
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
		"operation":  thingCreate,
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
		"operation":  thingUpdate,
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
		"operation":  thingUpdateGroupAndProfile,
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
		"operation": thingRemove,
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
		"operation": profileCreate,
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
		"operation": profileUpdate,
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
		"operation": profileRemove,
	}
}

type removeGroupEvent struct {
	id string
}

func (rge removeGroupEvent) Encode() map[string]any {
	return map[string]any{
		"id":        rge.id,
		"operation": groupRemove,
	}
}
