//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package http

import (
	"net/http"

	"github.com/mainflux/mainflux"
)

var _ mainflux.Response = (*infoRes)(nil)

type infoRes struct {
	Version       string `json:"version"`
	Os            string `json:"os"`
	UpTimeSeconds int    `json:"upTimeSeconds"`
}

func (res infoRes) Code() int {
	return http.StatusOK
}

func (res infoRes) Headers() map[string]string {
	return map[string]string{}
}

func (res infoRes) Empty() bool {
	return false
}

type createRes struct {
	Result string `json:"result"`
}

func (res createRes) Code() int {
	return http.StatusOK
}

func (res createRes) Headers() map[string]string {
	return map[string]string{}
}

func (res createRes) Empty() bool {
	return false
}
