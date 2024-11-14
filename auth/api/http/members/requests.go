package members

import (
	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

const maxNameSize = 254

type listMembersByOrgReq struct {
	token    string
	id       string
	offset   uint64
	limit    uint64
	metadata auth.OrgMetadata
}

func (req listMembersByOrgReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type memberReq struct {
	token    string
	orgID    string
	memberID string
}

func (req memberReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.orgID == "" || req.memberID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type membersReq struct {
	token      string
	orgID      string
	OrgMembers []auth.OrgMember `json:"org_members"`
}

func (req membersReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.orgID == "" {
		return apiutil.ErrMissingID
	}

	if len(req.OrgMembers) == 0 {
		return apiutil.ErrEmptyList
	}

	for _, m := range req.OrgMembers {
		if m.Role != auth.Admin && m.Role != auth.Viewer && m.Role != auth.Editor {
			return apiutil.ErrInvalidMemberRole
		}
	}

	return nil
}

type unassignMembersReq struct {
	token     string
	orgID     string
	MemberIDs []string `json:"member_ids"`
}

func (req unassignMembersReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.orgID == "" {
		return apiutil.ErrMissingID
	}

	if len(req.MemberIDs) == 0 {
		return apiutil.ErrEmptyList
	}

	return nil
}

type orgReq struct {
	token string
	id    string
}

func (req orgReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}
