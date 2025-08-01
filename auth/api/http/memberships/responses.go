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
	Name   string `json:"name"`
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

type viewFileRes struct {
	File     []byte
	FileName string
}

func (res viewFileRes) Code() int {
	return http.StatusOK
}

func (res viewFileRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewFileRes) Empty() bool {
	return len(res.File) == 0
}

type restoreRes struct{}

func (res restoreRes) Code() int {
	return http.StatusCreated
}

func (res restoreRes) Headers() map[string]string {
	return map[string]string{}
}

func (res restoreRes) Empty() bool {
	return true
}
