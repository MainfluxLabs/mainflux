package http

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/cron"
)

var (
	_ apiutil.Response = (*clientResponse)(nil)
	_ apiutil.Response = (*clientsRes)(nil)
	_ apiutil.Response = (*removeRes)(nil)
)

type pageRes struct {
	Total  uint64 `json:"total"`
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
	Order  string `json:"order,omitempty"`
	Dir    string `json:"dir,omitempty"`
	Name   string `json:"name,omitempty"`
}

type clientResponse struct {
	ID           string         `json:"id"`
	GroupID      string         `json:"group_id"`
	ThingID      string         `json:"thing_id"`
	Name         string         `json:"name"`
	IPAddress    string         `json:"ip_address"`
	Port         string         `json:"port"`
	SlaveID      uint8          `json:"slave_id"`
	FunctionCode string         `json:"function_code"`
	Scheduler    cron.Scheduler `json:"scheduler"`
	DataFields   []field        `json:"data_fields"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	updated      bool
}

func (res clientResponse) Code() int {
	return http.StatusOK
}

func (res clientResponse) Headers() map[string]string {
	return map[string]string{}
}

func (res clientResponse) Empty() bool {
	return res.updated
}

type clientsRes struct {
	Clients []clientResponse `json:"clients"`
	created bool
}

func (res clientsRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res clientsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res clientsRes) Empty() bool {
	return false
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

type clientsPageRes struct {
	pageRes
	Clients []clientResponse `json:"clients"`
}

func (res clientsPageRes) Code() int {
	return http.StatusOK
}

func (res clientsPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res clientsPageRes) Empty() bool {
	return false
}
