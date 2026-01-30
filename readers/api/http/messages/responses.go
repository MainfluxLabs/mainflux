// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package messages

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/readers"
)

var (
	_ apiutil.Response = (*listJSONMessagesRes)(nil)
	_ apiutil.Response = (*listSenMLMessagesRes)(nil)
	_ apiutil.Response = (*restoreMessagesRes)(nil)
)

type listJSONMessagesRes struct {
	readers.JSONPageMetadata
	Total    uint64            `json:"total"`
	Messages []readers.Message `json:"messages"`
}

func (res listJSONMessagesRes) Headers() map[string]string {
	return map[string]string{}
}

func (res listJSONMessagesRes) Code() int {
	return http.StatusOK
}

func (res listJSONMessagesRes) Empty() bool {
	return false
}

type listSenMLMessagesRes struct {
	readers.SenMLPageMetadata
	Total    uint64            `json:"total"`
	Messages []readers.Message `json:"messages"`
}

func (res listSenMLMessagesRes) Headers() map[string]string {
	return map[string]string{}
}

func (res listSenMLMessagesRes) Code() int {
	return http.StatusOK
}

func (res listSenMLMessagesRes) Empty() bool {
	return false
}

type restoreMessagesRes struct{}

func (res restoreMessagesRes) Code() int {
	return http.StatusCreated
}

func (res restoreMessagesRes) Headers() map[string]string {
	return map[string]string{}
}

func (res restoreMessagesRes) Empty() bool {
	return true
}

type exportFileRes struct {
	file []byte
}

func (res exportFileRes) Code() int {
	return http.StatusOK
}

func (res exportFileRes) Headers() map[string]string {
	return map[string]string{}
}

func (res exportFileRes) Empty() bool {
	return len(res.file) == 0
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
