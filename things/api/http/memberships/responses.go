// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package memberships

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

var (
	_ apiutil.Response = (*listGroupMembershipsRes)(nil)
)

type pageRes struct {
	Total  uint64 `json:"total"`
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
	Email  string `json:"email,omitempty"`
	Order  string `json:"order,omitempty"`
	Dir    string `json:"dir,omitempty"`
}

type groupMembership struct {
	MemberID string `json:"member_id"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

type ViewGroupMembershipRes struct {
	MemberID string `json:"member_id"`
	GroupID  string `json:"group_id"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

type listGroupMembershipsRes struct {
	pageRes
	GroupMemberships []groupMembership `json:"group_memberships"`
}

func (res listGroupMembershipsRes) Code() int {
	return http.StatusOK
}

func (res listGroupMembershipsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res listGroupMembershipsRes) Empty() bool {
	return false
}
