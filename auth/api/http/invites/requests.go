package invites

import (
	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

const (
	maxLimitSize = 200
	maxNameSize  = 254
)

type createOrgInviteReq struct {
	token        string
	orgID        string
	Email        string             `json:"email,omitempty"`
	Role         string             `json:"role,omitempty"`
	GroupInvites []auth.GroupInvite `json:"group_invites,omitempty"`
	RedirectPath string             `json:"redirect_path,omitempty"`
}

func (req createOrgInviteReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.orgID == "" {
		return apiutil.ErrMissingOrgID
	}

	if req.RedirectPath == "" {
		return apiutil.ErrMissingRedirectPath
	}

	if req.Email == "" {
		return apiutil.ErrMissingEmail
	}

	if err := apiutil.ValidateOrgInviteeRole(req.Role); err != nil {
		return err
	}

	if req.Role == auth.Owner {
		return errors.ErrMalformedEntity
	}

	for _, gi := range req.GroupInvites {
		if err := apiutil.ValidateOrgInviteeRole(gi.MemberRole); err != nil {
			return err
		}
	}

	return nil
}

type inviteReq struct {
	token string
	id    string
}

func (req inviteReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingInviteID
	}

	return nil
}

type respondOrgInviteReq struct {
	token    string
	id       string
	accepted bool
}

func (req respondOrgInviteReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingInviteID
	}

	return nil
}

type listOrgInvitesByUserReq struct {
	token string
	id    string
	pm    auth.PageMetadata
}

func (req listOrgInvitesByUserReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingUserID
	}

	return req.pm.Validate(maxLimitSize, maxNameSize)
}

type listOrgInvitesByOrgReq struct {
	token string
	id    string
	pm    auth.PageMetadata
}

func (req listOrgInvitesByOrgReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingOrgID
	}

	return req.pm.Validate(maxLimitSize, maxNameSize)
}
