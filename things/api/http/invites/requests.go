package invites

import (
	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/invites"
	"github.com/MainfluxLabs/mainflux/things"
)

const (
	maxLimitSize = 200
	maxNameSize  = 254
)

type createGroupInviteReq struct {
	token        string
	groupID      string
	Email        string `json:"email,omitempty"`
	Role         string `json:"role,omitempty"`
	RedirectPath string `json:"redirect_path,omitempty"`
}

func (req createGroupInviteReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingGroupID
	}

	if req.RedirectPath == "" {
		return apiutil.ErrMissingRedirectPath
	}

	if req.Email == "" {
		return apiutil.ErrMissingEmail
	}

	if err := validateRole(req.Role); err != nil {
		return err
	}

	if req.Role == auth.Owner {
		return apiutil.ErrMalformedEntity
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

type respondGroupInviteReq struct {
	token    string
	id       string
	accepted bool
}

func (req respondGroupInviteReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingInviteID
	}

	return nil
}

type listGroupInvitesByUserReq struct {
	token string
	id    string
	pm    invites.PageMetadataInvites
}

func (req listGroupInvitesByUserReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingUserID
	}

	if err := apiutil.ValidatePageMetadata(req.pm.PageMetadata, maxLimitSize, maxNameSize); err != nil {
		return err
	}

	return nil
}

type listGroupInvitesByGroupReq struct {
	token string
	id    string
	pm    invites.PageMetadataInvites
}

func (req listGroupInvitesByGroupReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingGroupID
	}

	if err := apiutil.ValidatePageMetadata(req.pm.PageMetadata, maxLimitSize, maxNameSize); err != nil {
		return err
	}

	return nil
}

type listGroupInvitesByOrgReq struct {
	token string
	id    string
	pm    invites.PageMetadataInvites
}

func (req listGroupInvitesByOrgReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingOrgID
	}

	if err := apiutil.ValidatePageMetadata(req.pm.PageMetadata, maxLimitSize, maxNameSize); err != nil {
		return err
	}

	return nil
}

func validateRole(role string) error {
	if role != things.Owner && role != things.Admin && role != things.Editor && role != things.Viewer {
		return apiutil.ErrInvalidRole
	}

	return nil
}
