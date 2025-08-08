package invites

import (
	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

const maxLimitSize = 200

type createInviteReq struct {
	token                string
	orgID                string
	OrgMember            auth.OrgMembership `json:"org_member,omitempty"`
	RedirectPathRegister string             `json:"redirect_path_register,omitempty"`
	RedirectPathInvite   string             `json:"redirect_path_invite,omitempty"`
}

func (req createInviteReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.orgID == "" {
		return apiutil.ErrMissingOrgID
	}

	if req.RedirectPathRegister == "" || req.RedirectPathInvite == "" {
		return apiutil.ErrMissingRedirectPath
	}

	if req.OrgMember.Email == "" {
		return apiutil.ErrMissingEmail
	}

	if req.OrgMember.Role != auth.Admin && req.OrgMember.Role != auth.Viewer && req.OrgMember.Role != auth.Editor {
		return apiutil.ErrInvalidRole
	}

	return nil
}

type inviteRevokeReq struct {
	token    string
	inviteID string
}

func (req inviteRevokeReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.inviteID == "" {
		return apiutil.ErrMissingInviteID
	}

	return nil
}

type inviteResponseReq struct {
	token          string
	inviteID       string
	inviteAccepted bool
}

func (req inviteResponseReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.inviteID == "" {
		return apiutil.ErrMissingInviteID
	}

	return nil
}

type listInvitesByUserReq struct {
	token  string
	userID string
	pm     apiutil.PageMetadata
}

func (req listInvitesByUserReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.userID == "" {
		return apiutil.ErrMissingUserID
	}

	if err := apiutil.ValidatePageMetadata(req.pm, maxLimitSize, 254); err != nil {
		return err
	}

	return nil
}

type viewInviteReq struct {
	token    string
	inviteID string
}

func (req viewInviteReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.inviteID == "" {
		return apiutil.ErrMissingInviteID
	}

	return nil
}
