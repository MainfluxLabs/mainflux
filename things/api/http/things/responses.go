// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package things

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

var (
	_ apiutil.Response = (*viewThingRes)(nil)
	_ apiutil.Response = (*thingsPageRes)(nil)
	_ apiutil.Response = (*ThingsPageRes)(nil)
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

type thingRes struct {
	ID          string         `json:"id"`
	GroupID     string         `json:"group_id,omitempty"`
	ProfileID   string         `json:"profile_id"`
	Name        string         `json:"name,omitempty"`
	Key         string         `json:"key"`
	ExternalKey string         `json:"external_key,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	created     bool
}

type thingsRes struct {
	Things  []thingRes `json:"things"`
	created bool
}

func (res thingsRes) Code() int {
	if res.created {
		return http.StatusCreated
	}

	return http.StatusOK
}

func (res thingsRes) Headers() map[string]string {
	return map[string]string{}
}

func (res thingsRes) Empty() bool {
	return false
}

type viewThingRes struct {
	ID          string         `json:"id"`
	GroupID     string         `json:"group_id,omitempty"`
	ProfileID   string         `json:"profile_id"`
	Name        string         `json:"name,omitempty"`
	Key         string         `json:"key"`
	ExternalKey string         `json:"external_key,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

func (res viewThingRes) Code() int {
	return http.StatusOK
}

func (res viewThingRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewThingRes) Empty() bool {
	return false
}

type viewMetadataRes struct {
	Metadata map[string]any `json:"metadata,omitempty"`
}

func (res viewMetadataRes) Code() int {
	return http.StatusOK
}

func (res viewMetadataRes) Headers() map[string]string {
	return map[string]string{}
}

func (res viewMetadataRes) Empty() bool {
	return false
}

type thingsPageRes struct {
	pageRes
	Things []viewThingRes `json:"things"`
}

func (res thingsPageRes) Code() int {
	return http.StatusOK
}

func (res thingsPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res thingsPageRes) Empty() bool {
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

type ThingsPageRes struct {
	pageRes
	Things []thingRes `json:"things"`
}

func (res ThingsPageRes) Code() int {
	return http.StatusOK
}

func (res ThingsPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res ThingsPageRes) Empty() bool {
	return false
}

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

type updateExternalKeyRes struct {
}

func (res updateExternalKeyRes) Code() int {
	return http.StatusCreated
}

func (res updateExternalKeyRes) Headers() map[string]string {
	return map[string]string{}
}

func (res updateExternalKeyRes) Empty() bool {
	return true
}
