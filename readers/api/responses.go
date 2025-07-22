// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/readers"
)

var (
	_ apiutil.Response = (*listMessagesRes)(nil)
	_ apiutil.Response = (*restoreMessagesRes)(nil)
)

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

type backupFileRes struct {
	file []byte
}

func (res backupFileRes) Code() int {
	return http.StatusOK
}

func (res backupFileRes) Headers() map[string]string {
	return map[string]string{}
}

func (res backupFileRes) Empty() bool {
	return false
}
