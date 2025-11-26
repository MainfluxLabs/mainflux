// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"net/http"
)

type alarmResponse struct {
	ID       string         `json:"id"`
	ThingID  string         `json:"thing_id"`
	GroupID  string         `json:"group_id"`
	RuleID   string         `json:"rule_id"`
	Subtopic string         `json:"subtopic"`
	Protocol string         `json:"protocol"`
	Payload  map[string]any `json:"payload"`
	Created  int64          `json:"created"`
}

type AlarmsPageRes struct {
	Total  uint64          `json:"total"`
	Offset uint64          `json:"offset"`
	Limit  uint64          `json:"limit"`
	Order  string          `json:"order,omitempty"`
	Dir    string          `json:"direction,omitempty"`
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
