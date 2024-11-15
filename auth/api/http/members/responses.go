package members

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

var (
	_ apiutil.Response = (*memberPageRes)(nil)
	_ apiutil.Response = (*assignRes)(nil)
	_ apiutil.Response = (*unassignRes)(nil)
)

type viewMemberRes struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

func (res viewMemberRes) Code() int {
	return http.StatusOK
}

func (res viewMemberRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewMemberRes) Empty() bool {
	return false
}

type memberPageRes struct {
	pageRes
	Members []viewMemberRes `json:"members"`
}

func (res memberPageRes) Code() int {
	return http.StatusOK
}

func (res memberPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res memberPageRes) Empty() bool {
	return false
}

type pageRes struct {
	Limit  uint64 `json:"limit"`
	Offset uint64 `json:"offset"`
	Total  uint64 `json:"total"`
	Name   string `json:"name"`
}

type assignRes struct{}

func (res assignRes) Code() int {
	return http.StatusOK
}

func (res assignRes) Headers() map[string]string {
	return map[string]string{}
}

func (res assignRes) Empty() bool {
	return true
}

type unassignRes struct{}

func (res unassignRes) Code() int {
	return http.StatusNoContent
}

func (res unassignRes) Headers() map[string]string {
	return map[string]string{}
}

func (res unassignRes) Empty() bool {
	return true
}
