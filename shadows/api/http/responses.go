// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/shadows"
)

var (
	_ apiutil.Response = (*shadowResponse)(nil)
	_ apiutil.Response = (*removeRes)(nil)
)

type stateRes struct {
	Desired  shadows.State `json:"desired"`
	Reported shadows.State `json:"reported"`
	Delta    shadows.State `json:"delta,omitempty"`
}

type shadowResponse struct {
	ThingID   string   `json:"thing_id"`
	State     stateRes `json:"state"`
	Timestamp int64    `json:"timestamp"`
}

func (res shadowResponse) Code() int {
	return http.StatusOK
}

func (res shadowResponse) Headers() map[string]string {
	return map[string]string{}
}

func (res shadowResponse) Empty() bool {
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

func buildShadowResponse(sh shadows.Shadow) shadowResponse {
	return shadowResponse{
		ThingID: sh.ThingID,
		State: stateRes{
			Desired:  sh.Desired,
			Reported: sh.Reported,
			Delta:    sh.Delta,
		},
		Timestamp: sh.Timestamp,
	}
}
