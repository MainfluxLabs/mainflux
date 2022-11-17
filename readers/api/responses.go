// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/readers"
)

var _ mainflux.Response = (*listMessagesPageRes)(nil)

type listMessagesPageRes struct {
	readers.PageMetadata
	Total    uint64            `json:"total"`
	Messages []readers.Message `json:"messages,omitempty"`
}

func (res listMessagesPageRes) Headers() map[string]string {
	return map[string]string{}
}

func (res listMessagesPageRes) Code() int {
	return http.StatusOK
}

func (res listMessagesPageRes) Empty() bool {
	return false
}

//type listAllMessagesRes struct {
//	Messages []readers.Message `json:"messages,omitempty"`
//}
//
//func (res listAllMessagesRes) Headers() map[string]string {
//	return map[string]string{}
//}
//
//func (res listAllMessagesRes) Code() int {
//	return http.StatusOK
//}
//
//func (res listAllMessagesRes) Empty() bool {
//	return false
//}
