// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package memberships

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/things"
)

const maxLimitSize = 200

type listGroupMembershipsReq struct {
	token        string
	groupID      string
	pageMetadata apiutil.PageMetadata
}

func (req listGroupMembershipsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingGroupID
	}

	if req.pageMetadata.Limit > maxLimitSize {
		return apiutil.ErrLimitSize
	}

	return nil
}

type createGroupMembershipsReq struct {
	token            string
	groupID          string
	GroupMemberships []groupMembership `json:"group_memberships"`
	RedirectPath     string            `json:"redirect_path,omitempty"`
}

func (req createGroupMembershipsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingGroupID
	}

	if len(req.GroupMemberships) == 0 {
		return apiutil.ErrEmptyList
	}

	for _, gm := range req.GroupMemberships {
		if gm.Role != things.Admin && gm.Role != things.Viewer && gm.Role != things.Editor {
			return apiutil.ErrInvalidRole
		}

		if gm.MemberID == "" {
			return apiutil.ErrMissingMemberID
		}
	}

	if req.RedirectPath == "" {
		return apiutil.ErrMissingRedirectPath
	}

	return nil
}

type updateGroupMembershipsReq struct {
	token            string
	groupID          string
	GroupMemberships []groupMembership `json:"group_memberships"`
}

func (req updateGroupMembershipsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingGroupID
	}

	if len(req.GroupMemberships) == 0 {
		return apiutil.ErrEmptyList
	}

	for _, gm := range req.GroupMemberships {
		if gm.Role != things.Admin && gm.Role != things.Viewer && gm.Role != things.Editor {
			return apiutil.ErrInvalidRole
		}

		if gm.MemberID == "" {
			return apiutil.ErrMissingMemberID
		}
	}

	return nil
}

type removeGroupMembershipsReq struct {
	token     string
	groupID   string
	MemberIDs []string `json:"member_ids"`
}

func (req removeGroupMembershipsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingGroupID
	}

	if len(req.MemberIDs) == 0 {
		return apiutil.ErrEmptyList
	}

	for _, id := range req.MemberIDs {
		if id == "" {
			return apiutil.ErrMissingMemberID
		}
	}

	return nil
}
