// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import "net/http"

type identityRes struct {
	ID string `json:"id"`
}

func (res identityRes) Code() int {
	return http.StatusOK
}

func (res identityRes) Headers() map[string]string {
	return map[string]string{}
}

func (res identityRes) Empty() bool {
	return false
}

type connByKeyRes struct {
	ChannelID string `json:"channel_id"`
	ThingID   string `json:"thing_id"`
}

func (res connByKeyRes) Code() int {
	return http.StatusOK
}

func (res connByKeyRes) Headers() map[string]string {
	return map[string]string{}
}

func (res connByKeyRes) Empty() bool {
	return false
}
