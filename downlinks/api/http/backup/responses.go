// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package backup

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

var (
	_ apiutil.Response = (*backupResponse)(nil)
	_ apiutil.Response = (*restoreRes)(nil)
)

type backupResponse struct {
	Downlinks []downlinkResponse `json:"downlinks"`
}

func (res backupResponse) Code() int {
	return http.StatusOK
}

func (res backupResponse) Headers() map[string]string {
	return map[string]string{}
}

func (res backupResponse) Empty() bool {
	return false
}

type downlinkResponse struct {
	ID         string            `json:"id"`
	GroupID    string            `json:"group_id"`
	ThingID    string            `json:"thing_id"`
	Name       string            `json:"name"`
	Url        string            `json:"url"`
	Method     string            `json:"method"`
	Payload    string            `json:"payload,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	Scheduler  schedulerRes      `json:"scheduler"`
	TimeFilter timeFilterRes     `json:"time_filter"`
	Metadata   map[string]any    `json:"metadata,omitempty"`
}

type schedulerRes struct {
	TimeZone  string  `json:"time_zone,omitempty"`
	Frequency string  `json:"frequency"`
	DateTime  string  `json:"date_time,omitempty"`
	Week      weekRes `json:"week,omitempty"`
	DayTime   string  `json:"day_time,omitempty"`
	Hour      int     `json:"hour,omitempty"`
	Minute    int     `json:"minute,omitempty"`
}

type weekRes struct {
	Days []string `json:"days,omitempty"`
	Time string   `json:"time,omitempty"`
}

type timeFilterRes struct {
	StartParam string `json:"start_param,omitempty"`
	EndParam   string `json:"end_param,omitempty"`
	Format     string `json:"format,omitempty"`
	Forecast   bool   `json:"forecast,omitempty"`
	Interval   string `json:"interval,omitempty"`
	Value      uint   `json:"value,omitempty"`
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
