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
	"github.com/mainflux/mainflux/re"
)

var _ mainflux.Response = (*infoRes)(nil)
var _ mainflux.Response = (*createRes)(nil)
var _ mainflux.Response = (*listRes)(nil)
var _ mainflux.Response = (*viewRes)(nil)

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

type listRes struct {
	Streams []string `json:"streams"`
}

func (res listRes) Code() int {
	return http.StatusOK
}

func (res listRes) Headers() map[string]string {
	return map[string]string{}
}

func (res listRes) Empty() bool {
	return false
}

type viewRes struct {
	Stream re.Stream
}

func (res viewRes) Code() int {
	return http.StatusOK
}

func (res viewRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewRes) Empty() bool {
	return false
}
