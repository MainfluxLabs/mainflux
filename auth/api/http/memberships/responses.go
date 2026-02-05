package memberships

import (
	"net/http"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

var (
	_ apiutil.Response = (*orgMembershipPageRes)(nil)
	_ apiutil.Response = (*createRes)(nil)
	_ apiutil.Response = (*removeRes)(nil)
)

type viewOrgMembershipRes struct {
	MemberID  string    `json:"member_id"`
	OrgID     string    `json:"org_id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (res viewOrgMembershipRes) Code() int {
	return http.StatusOK
}

func (res viewOrgMembershipRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewOrgMembershipRes) Empty() bool {
	return false
}

type orgMembershipPageRes struct {
	pageRes
	OrgMemberships []viewOrgMembershipRes `json:"org_memberships"`
}

func (res orgMembershipPageRes) Code() int {
	return http.StatusOK
}

func (res orgMembershipPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res orgMembershipPageRes) Empty() bool {
	return false
}

type pageRes struct {
	Limit  uint64 `json:"limit"`
	Offset uint64 `json:"offset"`
	Total  uint64 `json:"total"`
	Email  string `json:"email,omitempty"`
	Order  string `json:"order,omitempty"`
	Dir    string `json:"dir,omitempty"`
}

type createRes struct{}

func (res createRes) Code() int {
	return http.StatusOK
}

func (res createRes) Headers() map[string]string {
	return map[string]string{}
}

func (res createRes) Empty() bool {
	return true
}

type removeRes struct{}

func (res removeRes) Code() int {
	return http.StatusNoContent
}

func (res removeRes) Headers() map[string]string {
	return map[string]string{}
}

func (res removeRes) Empty() bool {
	return true
}
