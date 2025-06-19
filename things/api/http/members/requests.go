// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package members

import (
	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/things"
)

const maxLimitSize = 100

type listByGroupReq struct {
	token        string
	id           string
	pageMetadata apiutil.PageMetadata
}

func (req listByGroupReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingGroupID
	}

	if req.pageMetadata.Limit > maxLimitSize {
		return apiutil.ErrLimitSize
	}

	return nil
}

type groupMembersReq struct {
	token        string
	groupID      string
	GroupMembers []groupMember `json:"group_members"`
}

func (req groupMembersReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingGroupID
	}

	if len(req.GroupMembers) == 0 {
		return apiutil.ErrEmptyList
	}

	for _, gm := range req.GroupMembers {
		if gm.Role != auth.Admin && gm.Role != things.Viewer && gm.Role != things.Editor {
			return apiutil.ErrInvalidRole
		}

		if gm.ID == "" {
			return apiutil.ErrMissingMemberID
		}
	}

	return nil
}

type removeGroupMembersReq struct {
	token     string
	groupID   string
	MemberIDs []string `json:"member_ids"`
}

func (req removeGroupMembersReq) validate() error {
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
