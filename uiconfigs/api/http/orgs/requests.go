// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package orgs

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

const (
	maxLimitSize = 200
	maxNameSize  = 1024
)

type viewOrgConfigReq struct {
	token string
	orgID string
}

func (req *viewOrgConfigReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.orgID == "" {
		return apiutil.ErrMissingOrgID
	}

	return nil
}

type listOrgsConfigsReq struct {
	token        string
	pageMetadata apiutil.PageMetadata
}

func (req *listOrgsConfigsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	return apiutil.ValidatePageMetadata(req.pageMetadata, maxLimitSize, maxNameSize)
}

type updateOrgConfigReq struct {
	token  string
	orgID  string
	Config map[string]any `json:"config"`
}

func (req updateOrgConfigReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.orgID == "" {
		return apiutil.ErrMissingOrgID
	}

	return nil
}
