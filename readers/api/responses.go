// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/readers"
)

var _ mainflux.Response = (*listChannelMessagesPageRes)(nil)

type listChannelMessagesPageRes struct {
	readers.PageMetadata
	Total    uint64            `json:"total"`
	Messages []readers.Message `json:"messages,omitempty"`
}

func (res listChannelMessagesPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res listChannelMessagesPageRes) Code() int {
	return http.StatusOK
}

func (res listChannelMessagesPageRes) Empty() bool {
	return false
}

type listAllMessagesRes struct {
	Messages []readers.Message `json:"messages,omitempty"`
}

func (res listAllMessagesRes) Headers() map[string]string {
	return map[string]string{}
}

func (res listAllMessagesRes) Code() int {
	return http.StatusOK
}

func (res listAllMessagesRes) Empty() bool {
	return false
}
