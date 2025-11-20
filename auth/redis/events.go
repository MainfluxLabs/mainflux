package redis

import "encoding/json"

const (
	orgPrefix = "org."
	orgCreate = orgPrefix + "create"
	orgRemove = orgPrefix + "remove"
)

type event interface {
	Encode() map[string]interface{}
}

var (
	_ event = (*createOrgEvent)(nil)
	_ event = (*removeOrgEvent)(nil)
)

type createOrgEvent struct {
	id          string
	ownerID     string
	name        string
	description string
	metadata    map[string]interface{}
}

func (coe createOrgEvent) Encode() map[string]interface{} {
	val := map[string]interface{}{
		"id":        coe.id,
		"owner_id":  coe.ownerID,
		"operation": orgCreate,
	}

	if coe.name != "" {
		val["name"] = coe.name
	}

	if coe.description != "" {
		val["description"] = coe.description
	}

	if coe.metadata != nil {
		metadata, err := json.Marshal(coe.metadata)
		if err != nil {
			return val
		}

		val["metadata"] = string(metadata)
	}

	return val
}

type removeOrgEvent struct {
	id string
}

func (rte removeOrgEvent) Encode() map[string]interface{} {
	return map[string]interface{}{
		"id":        rte.id,
		"operation": orgRemove,
	}
}
