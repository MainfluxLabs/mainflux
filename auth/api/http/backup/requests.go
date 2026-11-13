package backup

import (
	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

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
	token          string
	Orgs           []viewOrgRes        `json:"orgs"`
	OrgMemberships []viewOrgMembership `json:"org_memberships"`
	UserRoles      []viewUserRole      `json:"user_roles"`
}

func (req restoreReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.Orgs) == 0 && len(req.OrgMemberships) == 0 && len(req.UserRoles) == 0 {
		return apiutil.ErrEmptyList
	}

	// User roles are validated here because orgs and memberships are saved before them.
	for _, ur := range req.UserRoles {
		if ur.UserID == "" {
			return apiutil.ErrMissingUserID
		}

		if ur.Role != auth.RoleRootAdmin && ur.Role != auth.RoleAdmin {
			return apiutil.ErrInvalidRole
		}
	}

	return nil
}
