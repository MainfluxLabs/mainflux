package invites

import (
	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

type invitesReq struct {
	token      string
	orgID      string
	OrgMembers []auth.OrgMember `json:"org_members"`
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

type revokeInviteReq struct {
	token    string
	orgID    string
	inviteID string
}

func (req revokeInviteReq) validate() error {
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
