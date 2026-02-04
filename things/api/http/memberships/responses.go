// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package memberships

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

var (
	_ apiutil.Response = (*removeRes)(nil)
	_ apiutil.Response = (*listGroupMembershipsRes)(nil)
	_ apiutil.Response = (*updateGroupMembershipsRes)(nil)
	_ apiutil.Response = (*createGroupMembershipsRes)(nil)
)

type removeRes struct{}

func (res removeRes) Code() int {
	return http.StatusNoContent
}

func (res removeRes) Headers() map[string]string {
	return map[string]string{}
}

func (res removeRes) Empty() bool {
	return true
}

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

type createGroupMembershipsRes struct{}

func (res createGroupMembershipsRes) Code() int {
	return http.StatusCreated
}

func (res createGroupMembershipsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res createGroupMembershipsRes) Empty() bool {
	return true
}

type updateGroupMembershipsRes struct{}

func (res updateGroupMembershipsRes) Code() int {
	return http.StatusOK
}

func (res updateGroupMembershipsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res updateGroupMembershipsRes) Empty() bool {
	return true
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
