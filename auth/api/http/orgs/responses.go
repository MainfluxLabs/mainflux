package orgs

import (
	"fmt"
	"net/http"
	"time"

	"github.com/MainfluxLabs/mainflux"
)

var (
	_ mainflux.Response = (*memberPageRes)(nil)
	_ mainflux.Response = (*orgRes)(nil)
	_ mainflux.Response = (*deleteRes)(nil)
	_ mainflux.Response = (*assignRes)(nil)
	_ mainflux.Response = (*unassignRes)(nil)
)

type memberPageRes struct {
	pageRes
	Members []string `json:"members"`
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

type shareOrgRes struct {
}

func (res shareOrgRes) Code() int {
	return http.StatusOK
}

func (res shareOrgRes) Headers() map[string]string {
	return map[string]string{}
}

func (res shareOrgRes) Empty() bool {
	return false
}

type viewOrgRes struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	OwnerID     string                 `json:"owner_id"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

func (res viewOrgRes) Code() int {
	return http.StatusOK
}

func (res viewOrgRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewOrgRes) Empty() bool {
	return false
}

type orgRes struct {
	id      string
	created bool
}

func (res orgRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res orgRes) Headers() map[string]string {
	if res.created {
		return map[string]string{
			"Location": fmt.Sprintf("/orgs/%s", res.id),
		}
	}

	return map[string]string{}
}

func (res orgRes) Empty() bool {
	return true
}

type orgPageRes struct {
	pageRes
	Orgs []viewOrgRes `json:"orgs"`
}

type pageRes struct {
	Limit  uint64 `json:"limit,omitempty"`
	Offset uint64 `json:"offset,omitempty"`
	Total  uint64 `json:"total"`
	Name   string `json:"name"`
}

func (res orgPageRes) Code() int {
	return http.StatusOK
}

func (res orgPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res orgPageRes) Empty() bool {
	return false
}

type deleteRes struct{}

func (res deleteRes) Code() int {
	return http.StatusNoContent
}

func (res deleteRes) Headers() map[string]string {
	return map[string]string{}
}

func (res deleteRes) Empty() bool {
	return true
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
