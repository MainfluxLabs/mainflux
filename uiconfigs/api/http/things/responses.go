package things

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

var (
	_ apiutil.Response = (*ThingConfigResponse)(nil)
	_ apiutil.Response = (*thingsConfigsRes)(nil)
)

type ThingConfigResponse struct {
	ThingID string         `json:"thing_id,omitempty"`
	GroupID string         `json:"group_id,omitempty"`
	Config  map[string]any `json:"config,omitempty"`
}

func (res ThingConfigResponse) Code() int {
	return http.StatusOK
}

func (res ThingConfigResponse) Headers() map[string]string {
	return map[string]string{}
}

func (res ThingConfigResponse) Empty() bool {
	return false
}

type thingsConfigsRes struct {
	Total         uint64                `json:"total"`
	Offset        uint64                `json:"offset"`
	Limit         uint64                `json:"limit"`
	ThingsConfigs []ThingConfigResponse `json:"things_configs"`
}

func (res thingsConfigsRes) Code() int {
	return http.StatusOK
}

func (res thingsConfigsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res thingsConfigsRes) Empty() bool {
	return false
}
