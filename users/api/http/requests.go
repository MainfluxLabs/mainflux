// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"regexp"

	"github.com/MainfluxLabs/mainflux/auth"
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

type registerByInviteReq struct {
	User         users.User `json:"user"`
	RedirectPath string     `json:"redirect_path"`
	inviteID     string
}

func (req registerByInviteReq) validate() error {
	if req.inviteID == "" {
		return apiutil.ErrMissingInviteID
	}

	if err := req.User.Validate(userPasswordRegex); err != nil {
		return err
	}

	if req.RedirectPath == "" {
		return apiutil.ErrMissingRedirectPath
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

type inviteReq struct {
	token    string
	inviteID string
}

func (req inviteReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.inviteID == "" {
		return apiutil.ErrMissingInviteID
	}

	return nil
}

type createPlatformInviteRequest struct {
	token        string
	Email        string                `json:"email,omitempty"`
	OrgID        string                `json:"org_id"`
	Role         string                `json:"role"`
	Groups       []auth.OrgInviteGroup `json:"groups"`
	RedirectPath string                `json:"redirect_path,omitempty"`
}

func (req createPlatformInviteRequest) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.Email == "" {
		return apiutil.ErrMissingEmail
	}

	if req.RedirectPath == "" {
		return apiutil.ErrMissingRedirectPath
	}

	if req.OrgID != "" && req.Role == "" {
		return apiutil.ErrMissingRole
	}

	return nil
}

type listPlatformInvitesRequest struct {
	token string
	pm    users.PageMetadataInvites
}

func (req listPlatformInvitesRequest) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if err := apiutil.ValidatePageMetadata(req.pm.PageMetadata, maxLimitSize, maxNameSize); err != nil {
		return err
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
	ID       string         `json:"id"`
	Email    string         `json:"email"`
	Password string         `json:"password"`
	Metadata map[string]any `json:"metadata"`
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
		return apiutil.ErrMissingUserID
	}

	return nil
}
