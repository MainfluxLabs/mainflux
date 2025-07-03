package memberships

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

var (
	_ apiutil.Response = (*membershipPageRes)(nil)
	_ apiutil.Response = (*createRes)(nil)
	_ apiutil.Response = (*removeRes)(nil)
)

type viewMembershipRes struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

func (res viewMembershipRes) Code() int {
	return http.StatusOK
}

func (res viewMembershipRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewMembershipRes) Empty() bool {
	return false
}

type membershipPageRes struct {
	pageRes
	Memberships []viewMembershipRes `json:"memberships"`
}

func (res membershipPageRes) Code() int {
	return http.StatusOK
}

func (res membershipPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res membershipPageRes) Empty() bool {
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
