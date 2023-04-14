package orgs

import (
	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/internal/apiutil"
)

const (
	adminRole = "admin"
	ownerRole = "owner"
	guestRole = "guest"
)

type createOrgReq struct {
	token       string
	Name        string                 `json:"name,omitempty"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

func (req createOrgReq) validate() error {
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
	metadata auth.OrgMetadata
}

func (req listOrgsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	return nil
}

type listOrgMembersReq struct {
	token    string
	id       string
	offset   uint64
	limit    uint64
	metadata auth.OrgMetadata
}

func (req listOrgMembersReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type listOrgGroupsReq struct {
	token    string
	id       string
	offset   uint64
	limit    uint64
	metadata auth.OrgMetadata
}

func (req listOrgGroupsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type listOrgMembershipsReq struct {
	token    string
	id       string
	offset   uint64
	limit    uint64
	metadata auth.OrgMetadata
}

func (req listOrgMembershipsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type assignMembersReq struct {
	token   string
	orgID   string
	Members []auth.Members `json:"members"`
}

func (req assignMembersReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.orgID == "" {
		return apiutil.ErrMissingID
	}

	if len(req.Members) == 0 {
		return apiutil.ErrEmptyList
	}

	for _, m := range req.Members {
		if m.Role != adminRole && m.Role != ownerRole && m.Role != guestRole && m.Role != "" {
			return apiutil.ErrInvalidIDFormat
		}
	}

	return nil
}

type unassignMembersReq struct {
	token        string
	orgID        string
	MemberEmails []string `json:"member_emails"`
}

func (req unassignMembersReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.orgID == "" {
		return apiutil.ErrMissingID
	}

	if len(req.MemberEmails) == 0 {
		return apiutil.ErrEmptyList
	}

	return nil
}

type groupsReq struct {
	token    string
	orgID    string
	GroupIDs []string `json:"group_ids"`
}

func (req groupsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.orgID == "" {
		return apiutil.ErrMissingID
	}

	if len(req.GroupIDs) == 0 {
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
