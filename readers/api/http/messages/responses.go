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

type searchJSONMessagesRes []searchJSONResultItem

type searchJSONResultItem struct {
	Total    uint64            `json:"total"`
	Messages []readers.Message `json:"messages"`
	Error    string            `json:"error,omitempty"`
}

func (res searchJSONMessagesRes) Headers() map[string]string {
	return map[string]string{}
}

func (res searchJSONMessagesRes) Code() int {
	errCount := 0
	for _, r := range res {
		if r.Error != "" {
			errCount++
		}
	}

	switch {
	case errCount == 0:
		return http.StatusOK
	case errCount < len(res):
		return http.StatusMultiStatus
	default:
		return http.StatusInternalServerError
	}
}

func (res searchJSONMessagesRes) Empty() bool {
	return false
}

type searchSenMLMessagesRes []searchSenMLResultItem

type searchSenMLResultItem struct {
	Total    uint64            `json:"total"`
	Messages []readers.Message `json:"messages"`
	Error    string            `json:"error,omitempty"`
}

func (res searchSenMLMessagesRes) Headers() map[string]string {
	return map[string]string{}
}

func (res searchSenMLMessagesRes) Code() int {
	errCount := 0
	for _, r := range res {
		if r.Error != "" {
			errCount++
		}
	}

	switch {
	case errCount == 0:
		return http.StatusOK
	case errCount < len(res):
		return http.StatusMultiStatus
	default:
		return http.StatusInternalServerError
	}
}

func (res searchSenMLMessagesRes) Empty() bool {
	return false
}
