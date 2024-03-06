// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"github.com/MainfluxLabs/mainflux"
	"net/http"
)

var _ mainflux.Response = (*pingRes)(nil)

type pingRes struct {
	Greeting string `json:"greeting"`
}

func (res pingRes) Code() int {
	return http.StatusOK
}

func (res pingRes) Headers() map[string]string {
	return map[string]string{}
}

func (res pingRes) Empty() bool {
	return false
}
