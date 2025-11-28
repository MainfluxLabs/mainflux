// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"fmt"
	"net/http"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

var (
	_ apiutil.Response = (*tokenRes)(nil)
	_ apiutil.Response = (*viewUserRes)(nil)
	_ apiutil.Response = (*passwChangeRes)(nil)
	_ apiutil.Response = (*createUserRes)(nil)
	_ apiutil.Response = (*deleteRes)(nil)
)

// MailSent message response when link is sent
const MailSent = "Email with reset link is sent"

type pageRes struct {
	Total  uint64 `json:"total"`
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
	Order  string `json:"order,omitempty"`
	Dir    string `json:"dir,omitempty"`
	Email  string `json:"email,omitempty"`
	Status string `json:"status,omitempty"`
}

type selfRegisterRes struct{}

func (res selfRegisterRes) Code() int {
	return http.StatusCreated
}

func (res selfRegisterRes) Headers() map[string]string {
	return map[string]string{}
}

func (res selfRegisterRes) Empty() bool {
	return true
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

type tokenRes struct {
	Token string `json:"token,omitempty"`
}

func (res tokenRes) Code() int {
	return http.StatusCreated
}

func (res tokenRes) Headers() map[string]string {
	return map[string]string{}
}

func (res tokenRes) Empty() bool {
	return res.Token == ""
}

type updateUserRes struct{}

func (res updateUserRes) Code() int {
	return http.StatusOK
}

func (res updateUserRes) Headers() map[string]string {
	return map[string]string{}
}

func (res updateUserRes) Empty() bool {
	return true
}

type viewUserRes struct {
	ID       string         `json:"id"`
	Email    string         `json:"email"`
	Metadata map[string]any `json:"metadata,omitempty"`
	Role     string         `json:"role,omitempty"`
}

func (res viewUserRes) Code() int {
	return http.StatusOK
}

func (res viewUserRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewUserRes) Empty() bool {
	return false
}

type userPageRes struct {
	pageRes
	Users []viewUserRes `json:"users"`
}

func (res userPageRes) Code() int {
	return http.StatusOK
}

func (res userPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res userPageRes) Empty() bool {
	return false
}

type passwResetReqRes struct {
	Msg string `json:"msg"`
}

func (res passwResetReqRes) Code() int {
	return http.StatusCreated
}

func (res passwResetReqRes) Headers() map[string]string {
	return map[string]string{}
}

func (res passwResetReqRes) Empty() bool {
	return false
}

type passwChangeRes struct {
}

func (res passwChangeRes) Code() int {
	return http.StatusCreated
}

func (res passwChangeRes) Headers() map[string]string {
	return map[string]string{}
}

func (res passwChangeRes) Empty() bool {
	return false
}

type deleteRes struct{}

func (res deleteRes) Code() int {
	return http.StatusNoContent
}

func (res deleteRes) Headers() map[string]string {
	return map[string]string{}
}

func (res deleteRes) Empty() bool {
	return true
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

type backupUserRes struct {
	ID       string         `json:"id"`
	Email    string         `json:"email"`
	Password string         `json:"password"`
	Metadata map[string]any `json:"metadata,omitempty"`
	Status   string         `json:"status"`
}

type backupRes struct {
	Users []backupUserRes `json:"users"`
	Admin backupUserRes   `json:"admin"`
}

func (res backupRes) Code() int {
	return http.StatusOK
}

func (res backupRes) Headers() map[string]string {
	return map[string]string{}
}

func (res backupRes) Empty() bool {
	return false
}

type restoreRes struct{}

func (res restoreRes) Code() int {
	return http.StatusCreated
}

func (res restoreRes) Headers() map[string]string {
	return map[string]string{}
}

func (res restoreRes) Empty() bool {
	return true
}
