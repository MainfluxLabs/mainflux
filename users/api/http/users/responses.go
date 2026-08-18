// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users

import (
	"fmt"
	"net/http"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

var (
	_ apiutil.Response = (*tokenRes)(nil)
	_ apiutil.Response = (*redirectURLRes)(nil)
	_ apiutil.Response = (*viewUserRes)(nil)
	_ apiutil.Response = (*passwChangeRes)(nil)
	_ apiutil.Response = (*createUserRes)(nil)
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
	Token        string    `json:"token,omitempty"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`

	// created distinguishes a fresh login (201) from a token refresh (200).
	created bool
}

func (res tokenRes) Code() int {
	if res.created {
		return http.StatusCreated
	}
	return http.StatusOK
}

func (res tokenRes) Headers() map[string]string {
	return map[string]string{}
}

func (res tokenRes) Empty() bool {
	return res.Token == ""
}

// oauthCallbackRes carries the refresh token alongside the redirect so the
// encoder can place it in an HttpOnly cookie rather than the URL fragment.
type oauthCallbackRes struct {
	RedirectURL  string
	RefreshToken string
}

type logoutRes struct{}

func (res logoutRes) Code() int { return http.StatusNoContent }

func (res logoutRes) Headers() map[string]string { return map[string]string{} }

func (res logoutRes) Empty() bool { return true }

type redirectURLRes struct {
	RedirectURL string `json:"url,omitempty"`
}

func (res redirectURLRes) Code() int {
	return http.StatusOK
}

func (res redirectURLRes) Headers() map[string]string {
	return map[string]string{}
}

func (res redirectURLRes) Empty() bool {
	return res.RedirectURL == ""
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

type oauthLoginRes struct {
	State        string `json:"state"`
	Verifier     string `json:"-"`
	InviteID     string `json:"-"`
	RedirectPath string `json:"-"`
	RedirectURL  string `json:"url"`
}

func (res oauthLoginRes) Code() int {
	return http.StatusOK
}

func (res oauthLoginRes) Headers() map[string]string {
	return map[string]string{}
}

func (res oauthLoginRes) Empty() bool {
	return false
}
