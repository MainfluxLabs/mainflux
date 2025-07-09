package invites

import "net/http"

type inviteRes struct{}

func (res inviteRes) Code() int {
	return http.StatusCreated
}

func (res inviteRes) Headers() map[string]string {
	return map[string]string{}
}

func (res inviteRes) Empty() bool {
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
