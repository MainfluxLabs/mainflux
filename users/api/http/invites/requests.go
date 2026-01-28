// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package invites

import (
	"regexp"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/users"
)

const (
	maxLimitSize = 200
	maxNameSize  = 254
)

var userPasswordRegex *regexp.Regexp

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
	Email        string             `json:"email,omitempty"`
	OrgID        string             `json:"org_id"`
	Role         string             `json:"role"`
	GroupInvites []auth.GroupInvite `json:"group_invites"`
	RedirectPath string             `json:"redirect_path,omitempty"`
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
