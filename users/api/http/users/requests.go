// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users

import (
	"regexp"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/users"
)

const (
	maxLimitSize = 200
	maxEmailSize = 1024
	maxNameSize  = 254
	EmailOrder   = "email"
	IDOrder      = "id"
	AscDir       = "asc"
	DescDir      = "desc"
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
	token    string
	status   string
	offset   uint64
	limit    uint64
	email    string
	metadata users.Metadata
	order    string
	dir      string
}

func (req listUsersReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.limit > maxLimitSize {
		return apiutil.ErrLimitSize
	}

	if len(req.email) > maxEmailSize {
		return apiutil.ErrEmailSize
	}

	if req.order != "" && req.order != EmailOrder && req.order != IDOrder {
		return apiutil.ErrInvalidOrder
	}

	if req.dir != "" && req.dir != AscDir && req.dir != DescDir {
		return apiutil.ErrInvalidDirection
	}

	if req.status != users.AllStatusKey &&
		req.status != users.EnabledStatusKey &&
		req.status != users.DisabledStatusKey {
		return apiutil.ErrInvalidStatus
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
		return users.ErrPasswordFormat
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
		return users.ErrPasswordFormat
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
