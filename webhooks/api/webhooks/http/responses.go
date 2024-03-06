// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"github.com/MainfluxLabs/mainflux"
	"net/http"
)

var _ mainflux.Response = (*webhookRes)(nil)

type webhookRes struct {
	created bool `json:"created"`
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
