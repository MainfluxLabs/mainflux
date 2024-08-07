// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

var (
	_ apiutil.Response = (*webhookResponse)(nil)
	_ apiutil.Response = (*webhooksRes)(nil)
	_ apiutil.Response = (*removeRes)(nil)
)

type webhookResponse struct {
	ID         string            `json:"id"`
	GroupID    string            `json:"group_id"`
	Name       string            `json:"name"`
	Url        string            `json:"url"`
	ResHeaders map[string]string `json:"headers"`
	updated    bool
}

func (res webhookResponse) Code() int {
	return http.StatusOK
}

func (res webhookResponse) Headers() map[string]string {
	return map[string]string{}
}

func (res webhookResponse) Empty() bool {
	if res.updated {
		return true
	}
	return false
}

type webhooksRes struct {
	Webhooks []webhookResponse `json:"webhooks"`
	created  bool
}

func (res webhooksRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res webhooksRes) Headers() map[string]string {
	return map[string]string{}
}

func (res webhooksRes) Empty() bool {
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
