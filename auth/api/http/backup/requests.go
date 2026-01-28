package backup

import (
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
}

func (req restoreReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.Orgs) == 0 && len(req.OrgMemberships) == 0 {
		return apiutil.ErrEmptyList
	}

	return nil
}
