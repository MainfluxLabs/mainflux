package memberships

import (
	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

const maxLimitSize = 200

type listOrgMembershipsReq struct {
	token  string
	orgID  string
	offset uint64
	limit  uint64
	email  string
	order  string
	dir    string
}

func (req listOrgMembershipsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.orgID == "" {
		return apiutil.ErrMissingOrgID
	}

	if req.limit > maxLimitSize {
		return apiutil.ErrLimitSize
	}

	return nil
}

type orgMembershipReq struct {
	token    string
	orgID    string
	memberID string
}

func (req orgMembershipReq) validate() error {
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

type orgMembershipsReq struct {
	token          string
	orgID          string
	OrgMemberships []auth.OrgMembership `json:"org_memberships"`
}

func (req orgMembershipsReq) validate() error {
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
		if err := auth.ValidateInviteeRole(m.Role); err != nil {
			return err
		}
	}

	return nil
}

type removeOrgMembershipsReq struct {
	token     string
	orgID     string
	MemberIDs []string `json:"member_ids"`
}

func (req removeOrgMembershipsReq) validate() error {
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
