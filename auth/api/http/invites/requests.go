package invites

import (
	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

const maxLimitSize = 100

type invitesReq struct {
	token      string
	orgID      string
	OrgMembers []auth.OrgMembership `json:"org_members"`
}

func (req invitesReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.orgID == "" {
		return apiutil.ErrMissingOrgID
	}

	if len(req.OrgMembers) == 0 {
		return apiutil.ErrEmptyList
	}

	for _, m := range req.OrgMembers {
		if m.Role != auth.Admin && m.Role != auth.Viewer && m.Role != auth.Editor {
			return apiutil.ErrInvalidRole
		}
	}

	return nil
}

type inviteRevokeReq struct {
	token    string
	orgID    string
	inviteID string
}

func (req inviteRevokeReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.orgID == "" {
		return apiutil.ErrMissingOrgID
	}

	if req.inviteID == "" {
		return apiutil.ErrMissingInviteID
	}

	return nil
}

type inviteResponseReq struct {
	token          string
	orgID          string
	inviteID       string
	inviteAccepted bool
}

func (req inviteResponseReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.orgID == "" {
		return apiutil.ErrMissingOrgID
	}

	if req.inviteID == "" {
		return apiutil.ErrMissingInviteID
	}

	return nil
}

type listInvitesByUserReq struct {
	token  string
	userID string
	offset uint64
	limit  uint64
}

func (req listInvitesByUserReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.userID == "" {
		return apiutil.ErrMissingUserID
	}

	if req.limit > maxLimitSize {
		return apiutil.ErrLimitSize
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
