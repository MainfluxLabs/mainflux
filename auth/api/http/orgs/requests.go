package orgs

import (
	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/internal/apiutil"
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

type assignOrgReq struct {
	token   string
	orgID   string
	Members []string `json:"members"`
}

func (req assignOrgReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.orgID == "" {
		return apiutil.ErrMissingID
	}

	if len(req.Members) == 0 {
		return apiutil.ErrEmptyList
	}

	return nil
}

type shareOrgAccessReq struct {
	token      string
	userOrgID  string
	ThingOrgID string `json:"thing_org_id"`
}

func (req shareOrgAccessReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.ThingOrgID == "" || req.userOrgID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type unassignOrgReq struct {
	assignOrgReq
}

func (req unassignOrgReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.orgID == "" {
		return apiutil.ErrMissingID
	}

	if len(req.Members) == 0 {
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
