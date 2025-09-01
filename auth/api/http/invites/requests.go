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
	token         string
	orgID         string
	OrgMembership auth.OrgMembership `json:"org_membership,omitempty"`
	RedirectPath  string             `json:"redirect_path,omitempty"`
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

	if req.OrgMembership.Email == "" {
		return apiutil.ErrMissingEmail
	}

	if req.OrgMembership.Role != auth.Admin && req.OrgMembership.Role != auth.Viewer && req.OrgMembership.Role != auth.Editor {
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
	token    string
	inviteID string
	accepted bool
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
	pm     auth.PageMetadataInvites
}

func (req listOrgInvitesByUserReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.userID == "" {
		return apiutil.ErrMissingUserID
	}

	if err := apiutil.ValidatePageMetadata(req.pm.PageMetadata, maxLimitSize, maxNameSize); err != nil {
		return err
	}

	return nil
}

type listOrgInvitesByOrgReq struct {
	token string
	orgID string
	pm    auth.PageMetadataInvites
}

func (req listOrgInvitesByOrgReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.orgID == "" {
		return apiutil.ErrMissingOrgID
	}

	if err := apiutil.ValidatePageMetadata(req.pm.PageMetadata, maxLimitSize, maxNameSize); err != nil {
		return err
	}

	return nil
}
