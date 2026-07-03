// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"github.com/MainfluxLabs/mainflux/audit"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

const maxLimitSize = 200

var allowedOrders = map[string]string{
	"id":          "id",
	"occurred_at": "occurred_at",
	"operation":   "operation",
	"actor_email": "actor_email",
	"org_id":      "org_id",
	"group_id":    "group_id",
}

// validatePageMetadata validates the audit page metadata.
func validatePageMetadata(pm audit.PageMetadata) error {
	common := apiutil.PageMetadata{Offset: pm.Offset, Limit: pm.Limit, Order: pm.Order, Dir: pm.Dir}

	return common.Validate(maxLimitSize, allowedOrders)
}

type listEventsReq struct {
	token        string
	pageMetadata audit.PageMetadata
}

func (req listEventsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	return validatePageMetadata(req.pageMetadata)
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

	return validatePageMetadata(req.pageMetadata)
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

	return validatePageMetadata(req.pageMetadata)
}
