// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package invites

import (
	"fmt"
	"net/http"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

var (
	_ apiutil.Response = (*createUserRes)(nil)
)

type pageRes struct {
	Total  uint64 `json:"total"`
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
	Order  string `json:"order,omitempty"`
	Dir    string `json:"dir,omitempty"`
	Email  string `json:"email,omitempty"`
	Status string `json:"status,omitempty"`
}

type createUserRes struct {
	ID      string
	created bool
}

func (res createUserRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res createUserRes) Headers() map[string]string {
	if res.created {
		return map[string]string{
			"Location": fmt.Sprintf("/users/%s", res.ID),
		}
	}

	return map[string]string{}
}

func (res createUserRes) Empty() bool {
	return true
}

type dormantOrgInvite struct {
	ID           string             `json:"id"`
	OrgID        string             `json:"org_id"`
	OrgName      string             `json:"org_name"`
	InviteeRole  string             `json:"invitee_role"`
	GroupInvites []auth.GroupInvite `json:"group_invites,omitempty"`
}

type platformInviteRes struct {
	ID           string            `json:"id,omitempty"`
	InviteeEmail string            `json:"invitee_email,omitempty"`
	CreatedAt    time.Time         `json:"created_at,omitempty"`
	ExpiresAt    time.Time         `json:"expires_at,omitempty"`
	State        string            `json:"state,omitempty"`
	OrgInvite    *dormantOrgInvite `json:"org_invite,omitempty"`
}

func (res platformInviteRes) Code() int {
	return http.StatusOK
}

func (res platformInviteRes) Headers() map[string]string {
	return map[string]string{}
}

func (res platformInviteRes) Empty() bool {
	return false
}

type createPlatformInviteRes struct {
	ID      string
	created bool
}

func (res createPlatformInviteRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res createPlatformInviteRes) Headers() map[string]string {
	if res.created {
		return map[string]string{
			"Location": fmt.Sprintf("/invites/%s", res.ID),
		}
	}

	return map[string]string{}
}

func (res createPlatformInviteRes) Empty() bool {
	return true
}

type platformInvitePageRes struct {
	pageRes
	Invites []platformInviteRes `json:"invites"`
}

type revokePlatformInviteRes struct{}

func (res revokePlatformInviteRes) Code() int {
	return http.StatusNoContent
}

func (res revokePlatformInviteRes) Headers() map[string]string {
	return map[string]string{}
}

func (res revokePlatformInviteRes) Empty() bool {
	return true
}
