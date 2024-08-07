package orgs

import (
	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

type createOrgsReq struct {
	token       string
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

func (req createOrgsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if len(req.Name) > maxNameSize || req.Name == "" {
		return apiutil.ErrNameSize
	}

	return nil
}

type updateOrgReq struct {
	token       string
	id          string
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

func (req updateOrgReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type listOrgsReq struct {
	token    string
	id       string
	name     string
	offset   uint64
	limit    uint64
	metadata auth.OrgMetadata
}

func (req listOrgsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	return nil
}

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

type listOrgsByMemberReq struct {
	token    string
	id       string
	name     string
	offset   uint64
	limit    uint64
	metadata auth.OrgMetadata
}

func (req listOrgsByMemberReq) validate() error {
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

type backupReq struct {
	token string
}

func (req backupReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	return nil
}

type restoreReq struct {
	token      string
	Orgs       []viewOrgRes     `json:"orgs"`
	OrgMembers []viewOrgMembers `json:"org_members"`
}

func (req restoreReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.Orgs) == 0 && len(req.OrgMembers) == 0 {
		return apiutil.ErrEmptyList
	}

	return nil
}
