// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package backup

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"

	orgs "github.com/MainfluxLabs/mainflux/uiconfigs/api/http/orgs"
	things "github.com/MainfluxLabs/mainflux/uiconfigs/api/http/things"
)

type backupReq struct {
	token string
}

func (req *backupReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	return nil
}

type restoreReq struct {
	token         string
	OrgsConfigs   []orgs.OrgConfigResponse     `json:"orgs_configs"`
	ThingsConfigs []things.ThingConfigResponse `json:"things_configs"`
}

func (req *restoreReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if len(req.OrgsConfigs) == 0 && len(req.ThingsConfigs) == 0 {
		return apiutil.ErrEmptyList
	}

	return nil
}
