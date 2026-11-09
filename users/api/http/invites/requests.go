// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package invites

import (
	"regexp"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/MainfluxLabs/mainflux/users/api"
)

const (
	maxLimitSize = 200
	maxEmailSize = 254
)

type registerByInviteReq struct {
	User          users.User `json:"user"`
	RedirectPath  string     `json:"redirect_path"`
	inviteID      string
	passwordRegex *regexp.Regexp
}

func (req registerByInviteReq) validate() error {
	if req.inviteID == "" {
		return apiutil.ErrMissingInviteID
	}

	if err := req.User.Validate(req.passwordRegex); err != nil {
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

type viewInviteReq struct {
	inviteID string
}

func (req viewInviteReq) validate() error {
	if req.inviteID == "" {
		return apiutil.ErrMissingInviteID
	}

	return nil
}

type createPlatformInviteRequest struct {
	token        string
	Email        string               `json:"email,omitempty"`
	OrgID        string               `json:"org_id"`
	Role         string               `json:"role"`
	GroupInvites []domain.GroupInvite `json:"group_invites"`
	RedirectPath string               `json:"redirect_path,omitempty"`
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

	if req.OrgID != "" {
		if err := apiutil.ValidateOrgInviteeRole(req.Role); err != nil {
			return err
		}

		// Org admins hold their group roles through their org role.
		if req.Role == domain.OrgAdmin && len(req.GroupInvites) > 0 {
			return errors.ErrMalformedEntity
		}
	}

	return nil
}

type listPlatformInvitesRequest struct {
	token string
	pm    users.PageMetadata
}

func (req listPlatformInvitesRequest) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	return api.ValidatePageMetadata(req.pm, maxLimitSize, maxEmailSize, users.PlatformInviteOrderFields)
}
