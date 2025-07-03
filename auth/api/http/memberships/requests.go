package memberships

import (
	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

const maxLimitSize = 100

type listMembershipsReq struct {
	token  string
	id     string
	offset uint64
	limit  uint64
	email  string
	order  string
	dir    string
}

func (req listMembershipsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingOrgID
	}

	if req.limit > maxLimitSize {
		return apiutil.ErrLimitSize
	}

	return nil
}

type membershipReq struct {
	token    string
	orgID    string
	memberID string
}

func (req membershipReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.orgID == "" {
		return apiutil.ErrMissingOrgID
	}

	if req.memberID == "" {
		return apiutil.ErrMissingMemberID
	}

	return nil
}

type membershipsReq struct {
	token          string
	orgID          string
	OrgMemberships []auth.OrgMembership `json:"org_memberships"`
}

func (req membershipsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.orgID == "" {
		return apiutil.ErrMissingOrgID
	}

	if len(req.OrgMemberships) == 0 {
		return apiutil.ErrEmptyList
	}

	for _, m := range req.OrgMemberships {
		if m.Role != auth.Admin && m.Role != auth.Viewer && m.Role != auth.Editor {
			return apiutil.ErrInvalidRole
		}
	}

	return nil
}

type removeMembershipsReq struct {
	token     string
	orgID     string
	MemberIDs []string `json:"member_ids"`
}

func (req removeMembershipsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.orgID == "" {
		return apiutil.ErrMissingOrgID
	}

	if len(req.MemberIDs) == 0 {
		return apiutil.ErrEmptyList
	}

	return nil
}
