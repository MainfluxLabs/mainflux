// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package groups

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

const (
	maxLimitSize = 100
	maxNameSize  = 1024
)

type createGroupReq struct {
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type createGroupsReq struct {
	token  string
	orgID  string
	Groups []createGroupReq
}

func (req createGroupsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.orgID == "" {
		return apiutil.ErrMissingOrgID
	}

	if len(req.Groups) <= 0 {
		return apiutil.ErrEmptyList
	}

	for _, group := range req.Groups {
		if group.Name == "" || len(group.Name) > maxNameSize {
			return apiutil.ErrNameSize
		}
	}

	return nil
}

type resourceReq struct {
	token string
	id    string
}

func (req resourceReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingGroupID
	}

	return nil
}

type viewByThingReq struct {
	token string
	id    string
}

func (req viewByThingReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingThingID
	}

	return nil
}

type viewByProfileReq struct {
	token string
	id    string
}

func (req viewByProfileReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingProfileID
	}

	return nil
}

type listReq struct {
	token        string
	pageMetadata apiutil.PageMetadata
}

func (req *listReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	return apiutil.ValidatePageMetadata(req.pageMetadata, maxLimitSize, maxNameSize)
}

type listByOrgReq struct {
	id           string
	token        string
	pageMetadata apiutil.PageMetadata
}

func (req listByOrgReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingOrgID
	}

	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	return apiutil.ValidatePageMetadata(req.pageMetadata, maxLimitSize, maxNameSize)
}

type updateGroupReq struct {
	token       string
	id          string
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

func (req updateGroupReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingGroupID
	}

	if req.Name == "" || len(req.Name) > maxNameSize {
		return apiutil.ErrNameSize
	}

	return nil
}

type removeGroupsReq struct {
	token    string
	GroupIDs []string `json:"group_ids,omitempty"`
}

func (req removeGroupsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.GroupIDs) < 1 {
		return apiutil.ErrEmptyList
	}

	for _, groupID := range req.GroupIDs {
		if groupID == "" {
			return apiutil.ErrMissingGroupID
		}
	}

	return nil
}
