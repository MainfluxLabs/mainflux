// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/readers"
)

var _ mainflux.Response = (*listMessagesRes)(nil)

type listMessagesRes struct {
	readers.PageMetadata
	Total    uint64            `json:"total"`
	Messages []readers.Message `json:"messages,omitempty"`
}

func (res listMessagesRes) Headers() map[string]string {
	return map[string]string{}
}

func (res listMessagesRes) Code() int {
	return http.StatusOK
}

func (res listMessagesRes) Empty() bool {
	return false
}
