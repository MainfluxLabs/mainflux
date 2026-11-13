// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/webrtc"
)

var (
	_ apiutil.Response = (*iceServersRes)(nil)
	_ apiutil.Response = (*viewRes)(nil)
)

type iceServersRes struct {
	ICEServers []webrtc.ICEServer `json:"ice_servers"`
}

func (res iceServersRes) Code() int {
	return http.StatusOK
}

func (res iceServersRes) Headers() map[string]string {
	return map[string]string{}
}

func (res iceServersRes) Empty() bool {
	return false
}

type viewRes struct {
	SessionID string `json:"session_id"`
	SDP       string `json:"sdp"`
}

func (res viewRes) Code() int {
	return http.StatusCreated
}

func (res viewRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewRes) Empty() bool {
	return false
}
