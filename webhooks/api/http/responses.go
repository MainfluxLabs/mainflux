// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux"
)

var _ mainflux.Response = (*webhookRes)(nil)

type webhookRes struct {
	created bool
}

func (res webhookRes) Code() int {
	if res.created {
		return http.StatusCreated
	}
	return http.StatusOK
}

func (res webhookRes) Headers() map[string]string {
	return map[string]string{}
}

func (res webhookRes) Empty() bool {
	return true
}

type webhookResponse struct {
	ThingID string `json:"thing_id"`
	Name    string `json:"name"`
	Format  string `json:"format"`
	Url     string `json:"url"`
}

type webhooksRes struct {
	Webhooks []webhookResponse `json:"webhooks"`
}

func (res webhooksRes) Code() int {
	return http.StatusOK
}

func (res webhooksRes) Headers() map[string]string {
	return map[string]string{}
}

func (res webhooksRes) Empty() bool {
	return false
}
