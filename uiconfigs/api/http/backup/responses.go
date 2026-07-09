package backup

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	orgs "github.com/MainfluxLabs/mainflux/uiconfigs/api/http/orgs"
	things "github.com/MainfluxLabs/mainflux/uiconfigs/api/http/things"
)

var (
	_ apiutil.Response = (*backupResponse)(nil)
)

type backupResponse struct {
	OrgsConfigs   []orgs.OrgConfigResponse     `json:"orgs_configs"`
	ThingsConfigs []things.ThingConfigResponse `json:"things_configs"`
}

func (res backupResponse) Code() int {
	return http.StatusOK
}

func (res backupResponse) Headers() map[string]string {
	return map[string]string{}
}

func (res backupResponse) Empty() bool {
	return false
}
