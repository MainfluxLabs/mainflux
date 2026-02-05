package orgs

import (
	"fmt"
	"net/http"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

var (
	_ apiutil.Response = (*orgRes)(nil)
	_ apiutil.Response = (*deleteRes)(nil)
)

type viewOrgRes struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	OwnerID     string         `json:"owner_id"`
	Description string         `json:"description,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
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

type orgsPageRes struct {
	pageRes
	Orgs []viewOrgRes `json:"orgs"`
}

type pageRes struct {
	Limit  uint64 `json:"limit"`
	Offset uint64 `json:"offset"`
	Total  uint64 `json:"total"`
	Ord    string `json:"order,omitempty"`
	Dir    string `json:"direction,omitempty"`
	Name   string `json:"name,omitempty"`
}

func (res orgsPageRes) Code() int {
	return http.StatusOK
}

func (res orgsPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res orgsPageRes) Empty() bool {
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
