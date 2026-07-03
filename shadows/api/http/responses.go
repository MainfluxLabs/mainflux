// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/shadows"
)

var (
	_ apiutil.Response = (*shadowRes)(nil)
	_ apiutil.Response = (*removeRes)(nil)
)

type stateRes struct {
	Desired  shadows.State `json:"desired"`
	Reported shadows.State `json:"reported"`
	Delta    shadows.State `json:"delta,omitempty"`
}

type shadowRes struct {
	ThingID    string   `json:"thing_id"`
	State      stateRes `json:"state"`
	ReportedAt int64    `json:"reported_at"`
	UpdatedAt  int64    `json:"updated_at"`
}

func (res shadowRes) Code() int {
	return http.StatusOK
}

func (res shadowRes) Headers() map[string]string {
	return map[string]string{}
}

func (res shadowRes) Empty() bool {
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

func buildShadowResponse(sh shadows.Shadow) shadowRes {
	return shadowRes{
		ThingID: sh.ThingID,
		State: stateRes{
			Desired:  sh.Desired,
			Reported: sh.Reported,
			Delta:    sh.Delta,
		},
		ReportedAt: sh.ReportedAt,
		UpdatedAt:  sh.UpdatedAt,
	}
}
