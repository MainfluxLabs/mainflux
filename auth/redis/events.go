package redis

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
	id string
}

func (coe createOrgEvent) Encode() map[string]interface{} {
	val := map[string]interface{}{
		"id":        coe.id,
		"operation": orgCreate,
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
