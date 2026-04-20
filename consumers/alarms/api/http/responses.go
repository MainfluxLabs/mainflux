// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux/consumers/alarms"
)

type alarmResponse struct {
	ID       string           `json:"id"`
	ThingID  string           `json:"thing_id"`
	GroupID  string           `json:"group_id"`
	RuleID   string           `json:"rule_id,omitempty"`
	ScriptID string           `json:"script_id,omitempty"`
	Subtopic string           `json:"subtopic"`
	Protocol string           `json:"protocol"`
	Rule     *alarms.RuleInfo `json:"rule,omitempty"`
	Level    int32            `json:"level"`
	Status   string           `json:"status"`
	Created  int64            `json:"created"`
}

type AlarmsPageRes struct {
	Total  uint64          `json:"total"`
	Offset uint64          `json:"offset"`
	Limit  uint64          `json:"limit"`
	Order  string          `json:"order,omitempty"`
	Dir    string          `json:"dir,omitempty"`
	Alarms []alarmResponse `json:"alarms"`
}

type updateAlarmStatusRes struct{}

func (res updateAlarmStatusRes) Code() int {
	return http.StatusOK
}

func (res updateAlarmStatusRes) Headers() map[string]string {
	return map[string]string{}
}

func (res updateAlarmStatusRes) Empty() bool {
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

type exportFileRes struct {
	file []byte
}

func (res exportFileRes) Code() int {
	return http.StatusOK
}

func (res exportFileRes) Headers() map[string]string {
	return map[string]string{}
}

func (res exportFileRes) Empty() bool {
	return len(res.file) == 0
}
