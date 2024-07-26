// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"net/http"
)

type notifierResponse struct {
	ID       string   `json:"id"`
	GroupID  string   `json:"group_id"`
	Contacts []string `json:"contacts"`
	updated  bool
}

func (res notifierResponse) Code() int {
	return http.StatusOK
}

func (res notifierResponse) Headers() map[string]string {
	return map[string]string{}
}

func (res notifierResponse) Empty() bool {
	return res.updated
}

type notifiersRes struct {
	Notifiers []notifierResponse `json:"notifiers"`
	created   bool
}

func (res notifiersRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res notifiersRes) Headers() map[string]string {
	return map[string]string{}
}

func (res notifiersRes) Empty() bool {
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
