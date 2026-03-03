package orgs

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

var (
	_ apiutil.Response = (*OrgConfigResponse)(nil)
	_ apiutil.Response = (*orgsConfigsRes)(nil)
	_ apiutil.Response = (*createRes)(nil)
	_ apiutil.Response = (*removeRes)(nil)
)

type OrgConfigResponse struct {
	OrgID  string         `json:"org_id,omitempty"`
	Config map[string]any `json:"config,omitempty"`
}

func (res OrgConfigResponse) Code() int {
	return http.StatusOK
}

func (res OrgConfigResponse) Headers() map[string]string {
	return map[string]string{}
}

func (res OrgConfigResponse) Empty() bool {
	return false
}

type orgsConfigsRes struct {
	Total       uint64              `json:"total"`
	Offset      uint64              `json:"offset"`
	Limit       uint64              `json:"limit"`
	OrgsConfigs []OrgConfigResponse `json:"orgs_configs"`
}

func (res orgsConfigsRes) Code() int {
	return http.StatusOK
}

func (res orgsConfigsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res orgsConfigsRes) Empty() bool {
	return false
}

type createRes struct{}

func (res createRes) Code() int {
	return http.StatusOK
}

func (res createRes) Headers() map[string]string {
	return map[string]string{}
}

func (res createRes) Empty() bool {
	return true
}

type removeRes struct{}

func (res removeRes) Code() int {
	return http.StatusNoContent
}

func (res removeRes) Headers() map[string]string {
	return map[string]string{}
}

func (res removeRes) Empty() bool {
	return true
}
