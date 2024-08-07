// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/users"
)

const (
	maxLimitSize = 100
	maxEmailSize = 1024
)

type userReq struct {
	user users.User
}

func (req userReq) validate() error {
	return req.user.Validate()
}

type selfRegisterUserReq struct {
	user users.User
}

func (req selfRegisterUserReq) validate() error {
	return req.user.Validate()
}

type registerUserReq struct {
	user  users.User
	token string
}

func (req registerUserReq) validate() error {
	if req.token == "" {
		return errors.ErrAuthorization
	}
	return req.user.Validate()
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
	if req.status != users.AllStatusKey &&
		req.status != users.EnabledStatusKey &&
		req.status != users.DisabledStatusKey {
		return apiutil.ErrInvalidStatus
	}

	return nil
}

type updateUserReq struct {
	token    string
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func (req updateUserReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	return nil
}

type passwResetReq struct {
	Email string `json:"email"`
	Host  string `json:"host"`
}

func (req passwResetReq) validate() error {
	if req.Email == "" {
		return apiutil.ErrMissingEmail
	}

	if req.Host == "" {
		return apiutil.ErrMissingHost
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
	OldPassword string `json:"old_password"`
}

func (req passwChangeReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.OldPassword == "" {
		return apiutil.ErrMissingPass
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
		return apiutil.ErrMissingID
	}
	return nil
}

type backupReq struct {
	token string
}

func (req backupReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	return nil
}

type restoreUserReq struct {
	ID       string                 `json:"id"`
	Email    string                 `json:"email"`
	Password string                 `json:"password"`
	Metadata map[string]interface{} `json:"metadata"`
	Status   string
}
type restoreReq struct {
	token string
	Users []restoreUserReq `json:"users"`
	Admin restoreUserReq   `json:"admin"`
}

func (req restoreReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.Users) == 0 {
		return apiutil.ErrEmptyList
	}

	if req.Admin.ID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}
