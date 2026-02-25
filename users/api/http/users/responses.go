// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users

import (
	"fmt"
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

var (
	_ apiutil.Response = (*tokenRes)(nil)
	_ apiutil.Response = (*redirectURLRes)(nil)
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

type oauthLoginRes struct {
	State       string `json:"state"`
	Verifier    string `json:"verifier"`
	RedirectURL string `json:"url"`
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
