package redis

import "github.com/MainfluxLabs/mainflux/pkg/events"

type event interface {
	Encode() map[string]any
}

var (
	_ event = (*createOrgEvent)(nil)
	_ event = (*removeOrgEvent)(nil)
)

type createOrgEvent struct {
	id string
}

func (coe createOrgEvent) Encode() map[string]any {
	val := map[string]any{
		"id":        coe.id,
		"operation": events.OrgCreate,
	}

	return val
}

type removeOrgEvent struct {
	id string
}

func (rte removeOrgEvent) Encode() map[string]any {
	return map[string]any{
		"id":        rte.id,
		"operation": events.OrgRemove,
	}
}
