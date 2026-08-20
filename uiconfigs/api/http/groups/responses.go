package groups

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

var (
	_ apiutil.Response = (*GroupConfigResponse)(nil)
	_ apiutil.Response = (*groupsConfigsRes)(nil)
)

type GroupConfigResponse struct {
	GroupID string         `json:"group_id,omitempty"`
	Config  map[string]any `json:"config,omitempty"`
}

func (res GroupConfigResponse) Code() int {
	return http.StatusOK
}

func (res GroupConfigResponse) Headers() map[string]string {
	return map[string]string{}
}

func (res GroupConfigResponse) Empty() bool {
	return false
}

type groupsConfigsRes struct {
	Total         uint64                `json:"total"`
	Offset        uint64                `json:"offset"`
	Limit         uint64                `json:"limit"`
	GroupsConfigs []GroupConfigResponse `json:"groups_configs"`
}

func (res groupsConfigsRes) Code() int {
	return http.StatusOK
}

func (res groupsConfigsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res groupsConfigsRes) Empty() bool {
	return false
}
