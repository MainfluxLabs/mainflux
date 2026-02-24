// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package backup

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

type backupReq struct {
	token string
}

func (req *backupReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	return nil
}

type restoreReq struct {
	token     string
	Downlinks []downlinkReq `json:"downlinks"`
}

func (req *restoreReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if len(req.Downlinks) == 0 {
		return apiutil.ErrEmptyList
	}

	return nil
}

type downlinkReq struct {
	ID         string            `json:"id"`
	GroupID    string            `json:"group_id"`
	ThingID    string            `json:"thing_id"`
	Name       string            `json:"name"`
	Url        string            `json:"url"`
	Method     string            `json:"method"`
	Payload    string            `json:"payload,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	Scheduler  schedulerReq      `json:"scheduler"`
	TimeFilter timeFilterReq     `json:"time_filter"`
	Metadata   map[string]any    `json:"metadata,omitempty"`
}

type schedulerReq struct {
	TimeZone  string  `json:"time_zone,omitempty"`
	Frequency string  `json:"frequency"`
	DateTime  string  `json:"date_time,omitempty"`
	Week      weekReq `json:"week,omitempty"`
	DayTime   string  `json:"day_time,omitempty"`
	Hour      int     `json:"hour,omitempty"`
	Minute    int     `json:"minute,omitempty"`
}

type weekReq struct {
	Days []string `json:"days,omitempty"`
	Time string   `json:"time,omitempty"`
}

type timeFilterReq struct {
	StartParam string `json:"start_param,omitempty"`
	EndParam   string `json:"end_param,omitempty"`
	Format     string `json:"format,omitempty"`
	Forecast   bool   `json:"forecast,omitempty"`
	Interval   string `json:"interval,omitempty"`
	Value      uint   `json:"value,omitempty"`
}
