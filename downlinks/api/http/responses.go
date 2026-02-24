package http

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux/downlinks"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/cron"
)

var (
	_ apiutil.Response = (*downlinkResponse)(nil)
	_ apiutil.Response = (*downlinksRes)(nil)
	_ apiutil.Response = (*removeRes)(nil)
)

type pageRes struct {
	Total  uint64 `json:"total"`
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
}

type downlinkResponse struct {
	ID         string               `json:"id"`
	GroupID    string               `json:"group_id"`
	ThingID    string               `json:"thing_id"`
	Name       string               `json:"name"`
	Url        string               `json:"url"`
	Method     string               `json:"method"`
	Payload    string               `json:"payload"`
	ResHeaders map[string]string    `json:"headers"`
	Scheduler  cron.Scheduler       `json:"scheduler"`
	TimeFilter downlinks.TimeFilter `json:"time_filter"`
	Metadata   map[string]any       `json:"metadata,omitempty"`
	updated    bool
}

func (res downlinkResponse) Code() int {
	return http.StatusOK
}

func (res downlinkResponse) Headers() map[string]string {
	return map[string]string{}
}

func (res downlinkResponse) Empty() bool {
	return res.updated
}

type downlinksRes struct {
	Downlinks []downlinkResponse `json:"downlinks"`
	created   bool
}

func (res downlinksRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res downlinksRes) Headers() map[string]string {
	return map[string]string{}
}

func (res downlinksRes) Empty() bool {
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

type downlinksPageRes struct {
	pageRes
	Downlinks []downlinkResponse `json:"downlinks"`
}

func (res downlinksPageRes) Code() int {
	return http.StatusOK
}

func (res downlinksPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res downlinksPageRes) Empty() bool {
	return false
}
