// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"net/http"
)

type alarmResponse struct {
	ID       string                 `json:"id"`
	ThingID  string                 `json:"thing_id"`
	GroupID  string                 `json:"group_id"`
	Subtopic string                 `json:"subtopic"`
	Protocol string                 `json:"protocol"`
	Rule     map[string]interface{} `json:"rule"`
	Payload  map[string]interface{} `json:"payload"`
	Created  int64                  `json:"created"`
}

type AlarmsPageRes struct {
	Total  uint64          `json:"total"`
	Offset uint64          `json:"offset"`
	Limit  uint64          `json:"limit"`
	Alarms []alarmResponse `json:"alarms"`
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
