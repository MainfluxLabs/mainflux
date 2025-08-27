package invites

import (
	"fmt"
	"net/http"
	"time"
)

type createOrgInviteRes struct {
	ID      string
	created bool
}

func (res createOrgInviteRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res createOrgInviteRes) Headers() map[string]string {
	if res.created {
		return map[string]string{
			"Location": fmt.Sprintf("/invites/%s", res.ID),
		}
	}

	return map[string]string{}
}

func (res createOrgInviteRes) Empty() bool {
	return true
}

type revokeOrgInviteRes struct{}

func (res revokeOrgInviteRes) Code() int {
	return http.StatusNoContent
}

func (res revokeOrgInviteRes) Headers() map[string]string {
	return map[string]string{}
}

func (res revokeOrgInviteRes) Empty() bool {
	return true
}

type respondOrgInviteRes struct {
	accept bool
}

func (res respondOrgInviteRes) Code() int {
	if res.accept {
		return http.StatusCreated
	} else {
		return http.StatusNoContent
	}
}

func (res respondOrgInviteRes) Headers() map[string]string {
	return map[string]string{}
}

func (res respondOrgInviteRes) Empty() bool {
	return true
}

type orgInviteRes struct {
	ID          string    `json:"id"`
	InviteeID   string    `json:"invitee_id"`
	InviterID   string    `json:"inviter_id"`
	OrgID       string    `json:"org_id"`
	InviteeRole string    `json:"invitee_role"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	State       string    `json:"state"`
}

func (res orgInviteRes) Code() int {
	return http.StatusOK
}

func (res orgInviteRes) Headers() map[string]string {
	return map[string]string{}
}

func (res orgInviteRes) Empty() bool {
	return false
}

type pageRes struct {
	Limit  uint64 `json:"limit"`
	Offset uint64 `json:"offset"`
	Total  uint64 `json:"total"`
}

type orgInvitePageRes struct {
	pageRes
	Invites []orgInviteRes `json:"invites"`
}

func (res orgInvitePageRes) Code() int {
	return http.StatusOK
}

func (res orgInvitePageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res orgInvitePageRes) Empty() bool {
	return false
}
