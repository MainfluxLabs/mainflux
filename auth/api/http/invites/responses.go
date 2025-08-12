package invites

import (
	"net/http"
	"time"
)

type createInviteRes struct{}

func (res createInviteRes) Code() int {
	return http.StatusCreated
}

func (res createInviteRes) Headers() map[string]string {
	return map[string]string{}
}

func (res createInviteRes) Empty() bool {
	return true
}

type revokeInviteRes struct{}

func (res revokeInviteRes) Code() int {
	return http.StatusNoContent
}

func (res revokeInviteRes) Headers() map[string]string {
	return map[string]string{}
}

func (res revokeInviteRes) Empty() bool {
	return true
}

type respondInviteRes struct{}

func (res respondInviteRes) Code() int {
	return http.StatusOK
}

func (res respondInviteRes) Headers() map[string]string {
	return map[string]string{}
}

func (res respondInviteRes) Empty() bool {
	return true
}

type inviteRes struct {
	ID           string    `json:"id"`
	InviteeID    string    `json:"invitee_id"`
	InviteeEmail string    `json:"invitee_email"`
	InviterID    string    `json:"inviter_id"`
	OrgID        string    `json:"org_id"`
	InviteeRole  string    `json:"invitee_role"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

func (res inviteRes) Code() int {
	return http.StatusOK
}

func (res inviteRes) Headers() map[string]string {
	return map[string]string{}
}

func (res inviteRes) Empty() bool {
	return false
}

type pageRes struct {
	Limit  uint64 `json:"limit"`
	Offset uint64 `json:"offset"`
	Total  uint64 `json:"total"`
}

type invitePageRes struct {
	pageRes
	Invites []inviteRes `json:"invites"`
}

func (res invitePageRes) Code() int {
	return http.StatusOK
}

func (res invitePageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res invitePageRes) Empty() bool {
	return false
}
