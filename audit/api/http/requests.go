// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"github.com/MainfluxLabs/mainflux/audit"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

const maxLimitSize = 200

type listEventsReq struct {
	token        string
	pageMetadata audit.PageMetadata
}

func (req listEventsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	return req.pageMetadata.Validate(maxLimitSize)
}

type listEventsByOrgReq struct {
	orgID        string
	token        string
	pageMetadata audit.PageMetadata
}

func (req listEventsByOrgReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.orgID == "" {
		return apiutil.ErrMissingOrgID
	}

	return req.pageMetadata.Validate(maxLimitSize)
}

type listEventsByGroupReq struct {
	groupID      string
	token        string
	pageMetadata audit.PageMetadata
}

func (req listEventsByGroupReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingGroupID
	}

	return req.pageMetadata.Validate(maxLimitSize)
}
