// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"net/http"
)

type notifierResponse struct {
	ID       string                 `json:"id"`
	GroupID  string                 `json:"group_id"`
	Name     string                 `json:"name"`
	Contacts []string               `json:"contacts"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	updated  bool
}

type pageRes struct {
	Total  uint64 `json:"total"`
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
	Order  string `json:"order"`
	Dir    string `json:"direction"`
	Name   string `json:"name"`
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

type NotifiersPageRes struct {
	pageRes
	Notifiers []notifierResponse `json:"notifiers"`
}

func (res NotifiersPageRes) Code() int {
	return http.StatusOK
}

func (res NotifiersPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res NotifiersPageRes) Empty() bool {
	return false
}
