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
	name     string
	offset   uint64
	limit    uint64
	admin    bool
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
	name     string
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
	token   string
	orgID   string
	Members []auth.Member `json:"members"`
}

func (req membersReq) validate() error {
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
		if m.Role != auth.AdminRole && m.Role != auth.ViewerRole && m.Role != auth.EditorRole {
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

type memberPolicy struct {
	MemberID string `json:"member_id"`
	Policy   string `json:"policy"`
}

type updatePolicyReq struct {
	token   string
	orgID   string
	groupID string
	Members []memberPolicy `json:"members"`
}

func (req updatePolicyReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.orgID == "" || req.groupID == "" {
		return apiutil.ErrMissingID
	}

	if len(req.Members) == 0 {
		return apiutil.ErrEmptyList
	}

	for _, m := range req.Members {
		if m.Policy != auth.RPolicy && m.Policy != auth.RwPolicy {
			return apiutil.ErrInvalidPolicy
		}

		if m.MemberID == "" {
			return apiutil.ErrMissingID
		}
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
	token           string
	Orgs            []auth.Org            `json:"orgs"`
	MemberRelations []auth.MemberRelation `json:"member_relations"`
	GroupRelations  []auth.GroupRelation  `json:"group_relations"`
}

func (req restoreReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.Orgs) == 0 && len(req.MemberRelations) == 0 && len(req.GroupRelations) == 0 {
		return apiutil.ErrEmptyList
	}

	return nil
}
