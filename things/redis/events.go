package redis

import "encoding/json"

const (
	thingPrefix = "thing."
	thingCreate = thingPrefix + "create"
	thingUpdate = thingPrefix + "update"
	thingRemove = thingPrefix + "remove"

	profilePrefix = "profile."
	profileCreate = profilePrefix + "create"
	profileUpdate = profilePrefix + "update"
	profileRemove = profilePrefix + "remove"
)

type event interface {
	Encode() map[string]interface{}
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
	metadata  map[string]interface{}
}

func (cte createThingEvent) Encode() map[string]interface{} {
	val := map[string]interface{}{
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
	metadata  map[string]interface{}
}

func (ute updateThingEvent) Encode() map[string]interface{} {
	val := map[string]interface{}{
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

type removeThingEvent struct {
	id string
}

func (rte removeThingEvent) Encode() map[string]interface{} {
	return map[string]interface{}{
		"id":        rte.id,
		"operation": thingRemove,
	}
}

type createProfileEvent struct {
	id       string
	groupID  string
	name     string
	metadata map[string]interface{}
}

func (cce createProfileEvent) Encode() map[string]interface{} {
	val := map[string]interface{}{
		"id":        cce.id,
		"group_id":  cce.groupID,
		"operation": profileCreate,
	}

	if cce.name != "" {
		val["name"] = cce.name
	}

	if cce.metadata != nil {
		metadata, err := json.Marshal(cce.metadata)
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
	metadata map[string]interface{}
}

func (uce updateProfileEvent) Encode() map[string]interface{} {
	val := map[string]interface{}{
		"id":        uce.id,
		"operation": profileUpdate,
	}

	if uce.name != "" {
		val["name"] = uce.name
	}

	if uce.metadata != nil {
		metadata, err := json.Marshal(uce.metadata)
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

func (rce removeProfileEvent) Encode() map[string]interface{} {
	return map[string]interface{}{
		"id":        rce.id,
		"operation": profileRemove,
	}
}
