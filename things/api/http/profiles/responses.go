// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package profiles

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

var (
	_ apiutil.Response = (*profilesPageRes)(nil)
	_ apiutil.Response = (*profilesRes)(nil)
	_ apiutil.Response = (*removeRes)(nil)
)

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

type profileRes struct {
	ID       string                 `json:"id"`
	GroupID  string                 `json:"group_id,omitempty"`
	Name     string                 `json:"name,omitempty"`
	Config   map[string]interface{} `json:"config,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	created  bool
}

type profilesRes struct {
	Profiles []profileRes `json:"profiles"`
	created  bool
}

func (res profilesRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res profilesRes) Headers() map[string]string {
	return map[string]string{}
}

func (res profilesRes) Empty() bool {
	return false
}

type profilesPageRes struct {
	pageRes
	Profiles []profileRes `json:"profiles"`
}

func (res profilesPageRes) Code() int {
	return http.StatusOK
}

func (res profilesPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res profilesPageRes) Empty() bool {
	return false
}

type pageRes struct {
	Total  uint64 `json:"total"`
	Offset uint64 `json:"offset"`
	Limit  uint64 `json:"limit"`
	Order  string `json:"order,omitempty"`
	Dir    string `json:"direction,omitempty"`
	Name   string `json:"name,omitempty"`
}
