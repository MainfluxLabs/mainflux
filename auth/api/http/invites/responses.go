package invites

import (
	"fmt"
	"net/http"
	"time"
)

type createOrgInviteRes struct{}

func (res createOrgInviteRes) Code() int {
	return http.StatusCreated
}

func (res createOrgInviteRes) Headers() map[string]string {
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

type platformInviteRes struct {
	ID           string    `json:"id,omitempty"`
	InviteeEmail string    `json:"invitee_email,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	State        string    `json:"state,omitempty"`
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

type createdPlatformInviteRes struct {
	ID      string
	created bool
}

func (res createdPlatformInviteRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res createdPlatformInviteRes) Headers() map[string]string {
	if res.created {
		return map[string]string{
			"Location": fmt.Sprintf("/invites-platform/%s", res.ID),
		}
	}

	return map[string]string{}
}

func (res createdPlatformInviteRes) Empty() bool {
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
