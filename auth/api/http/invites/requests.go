package invites

import (
	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

const (
	maxLimitSize = 200
	maxNameSize  = 254
)

type createOrgInviteReq struct {
	token        string
	orgID        string
	OrgMember    auth.OrgMembership `json:"org_member,omitempty"`
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

	if req.OrgMember.Email == "" {
		return apiutil.ErrMissingEmail
	}

	if req.OrgMember.Role != auth.Admin && req.OrgMember.Role != auth.Viewer && req.OrgMember.Role != auth.Editor {
		return apiutil.ErrInvalidRole
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

type orgInviteResponseReq struct {
	token          string
	inviteID       string
	inviteAccepted bool
}

func (req orgInviteResponseReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.inviteID == "" {
		return apiutil.ErrMissingInviteID
	}

	return nil
}

type listOrgInvitesByUserReq struct {
	token  string
	userID string
	pm     apiutil.PageMetadata
}

func (req listOrgInvitesByUserReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.userID == "" {
		return apiutil.ErrMissingUserID
	}

	if err := apiutil.ValidatePageMetadata(req.pm, maxLimitSize, maxNameSize); err != nil {
		return err
	}

	return nil
}

type listOrgInvitesByOrgReq struct {
	token string
	orgID string
	pm    apiutil.PageMetadata
}

func (req listOrgInvitesByOrgReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.orgID == "" {
		return apiutil.ErrMissingOrgID
	}

	if err := apiutil.ValidatePageMetadata(req.pm, maxLimitSize, maxNameSize); err != nil {
		return err
	}

	return nil
}
