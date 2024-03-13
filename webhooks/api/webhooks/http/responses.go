// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"github.com/MainfluxLabs/mainflux"
	"net/http"
)

var _ mainflux.Response = (*webhookRes)(nil)

type webhookRes struct {
	Created bool `json:"created"`
}

func (res webhookRes) Code() int {

	if res.Created {
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
	ThingID string `json:"thingID"`
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
	return true
}
