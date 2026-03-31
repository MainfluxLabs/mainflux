// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users

import (
	"regexp"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/MainfluxLabs/mainflux/users/api"
)

const (
	maxLimitSize = 200
	maxNameSize  = 254
	maxEmailSize = 254
)

var userPasswordRegex *regexp.Regexp

type userReq struct {
	user users.User
}

func (req userReq) validate() error {
	return req.user.Validate(userPasswordRegex)
}

type selfRegisterUserReq struct {
	User         users.User `json:"user"`
	RedirectPath string     `json:"redirect_path"`
}

func (req selfRegisterUserReq) validate() error {
	if req.RedirectPath == "" {
		return apiutil.ErrMissingRedirectPath
	}

	return req.User.Validate(userPasswordRegex)
}

type oauthLoginReq struct {
	provider     string
	inviteID     string
	redirectPath string
}

func (req oauthLoginReq) validate() error {
	if req.provider != users.GoogleProvider && req.provider != users.GitHubProvider {
		return apiutil.ErrInvalidProvider
	}
	return nil
}

type oauthCallbackReq struct {
	provider      string
	code          string
	state         string
	originalState string
	verifier      string
	inviteID      string
	redirectPath  string
}

func (req oauthCallbackReq) validate() error {
	if req.provider != users.GoogleProvider && req.provider != users.GitHubProvider {
		return apiutil.ErrInvalidProvider
	}

	if req.code == "" {
		return apiutil.ErrMissingProviderCode
	}

	if req.state == "" || req.originalState == "" || req.state != req.originalState {
		return apiutil.ErrInvalidState
	}

	return nil
}

type verifyEmailReq struct {
	emailToken string
}

func (req verifyEmailReq) validate() error {
	if req.emailToken == "" {
		return apiutil.ErrMissingEmailToken
	}

	return nil
}

type registerUserReq struct {
	user  users.User
	token string
}

func (req registerUserReq) validate() error {
	if req.token == "" {
		return errors.ErrAuthorization
	}
	return req.user.Validate(userPasswordRegex)
}

type viewUserReq struct {
	token string
	id    string
}

func (req viewUserReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	return nil
}

type listUsersReq struct {
	token string
	pm    users.PageMetadata
}

func (req listUsersReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if err := api.ValidatePageMetadata(req.pm, maxLimitSize, maxEmailSize); err != nil {
		return err
	}

	return nil
}

type updateUserReq struct {
	token    string
	Metadata map[string]any `json:"metadata,omitempty"`
}

func (req updateUserReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	return nil
}

type passwResetReq struct {
	Email        string `json:"email"`
	RedirectPath string `json:"redirect_path"`
}

func (req passwResetReq) validate() error {
	if req.Email == "" {
		return apiutil.ErrMissingEmail
	}

	if req.RedirectPath == "" {
		return apiutil.ErrMissingRedirectPath
	}

	return nil
}

type resetTokenReq struct {
	Token    string `json:"token"`
	Password string `json:"password"`
	ConfPass string `json:"confirm_password"`
}

func (req resetTokenReq) validate() error {
	if req.Password == "" {
		return apiutil.ErrMissingPass
	}

	if !userPasswordRegex.MatchString(req.Password) {
		return errors.ErrPasswordFormat
	}

	if req.ConfPass == "" {
		return apiutil.ErrMissingConfPass
	}

	if req.Token == "" {
		return apiutil.ErrBearerToken
	}

	if req.Password != req.ConfPass {
		return apiutil.ErrInvalidResetPass
	}

	return nil
}

type passwChangeReq struct {
	token       string
	Password    string `json:"password"`
	OldPassword string `json:"old_password,omitempty"`
	Email       string `json:"email,omitempty"`
}

func (req passwChangeReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.Email == "" && req.OldPassword == "" {
		return apiutil.ErrMissingPass
	}

	if !userPasswordRegex.MatchString(req.Password) {
		return errors.ErrPasswordFormat
	}

	return nil
}

type changeUserStatusReq struct {
	token string
	id    string
}

func (req changeUserStatusReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.id == "" {
		return apiutil.ErrMissingUserID
	}
	return nil
}
