// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package groups

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/uiconfigs/api"
)

const (
	maxLimitSize = 200
)

type viewGroupConfigReq struct {
	token   string
	groupID string
}

func (req *viewGroupConfigReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingGroupID
	}

	return nil
}

type listGroupsConfigsReq struct {
	token        string
	pageMetadata apiutil.PageMetadata
}

func (req *listGroupsConfigsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	return api.ValidatePageMetadata(req.pageMetadata, maxLimitSize)
}

type updateGroupConfigReq struct {
	token   string
	groupID string
	Config  map[string]any `json:"config"`
}

func (req updateGroupConfigReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingGroupID
	}

	return nil
}
