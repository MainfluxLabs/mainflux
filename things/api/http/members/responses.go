// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package members

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

var (
	_ apiutil.Response = (*removeRes)(nil)
	_ apiutil.Response = (*listGroupMembersRes)(nil)
	_ apiutil.Response = (*updateGroupMembersRes)(nil)
	_ apiutil.Response = (*createGroupMembersRes)(nil)
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
}

type groupMember struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type createGroupMembersRes struct{}

func (res createGroupMembersRes) Code() int {
	return http.StatusCreated
}

func (res createGroupMembersRes) Headers() map[string]string {
	return map[string]string{}
}

func (res createGroupMembersRes) Empty() bool {
	return true
}

type updateGroupMembersRes struct{}

func (res updateGroupMembersRes) Code() int {
	return http.StatusOK
}

func (res updateGroupMembersRes) Headers() map[string]string {
	return map[string]string{}
}

func (res updateGroupMembersRes) Empty() bool {
	return true
}

type listGroupMembersRes struct {
	pageRes
	GroupMembers []groupMember `json:"group_members"`
}

func (res listGroupMembersRes) Code() int {
	return http.StatusOK
}

func (res listGroupMembersRes) Headers() map[string]string {
	return map[string]string{}
}

func (res listGroupMembersRes) Empty() bool {
	return false
}
