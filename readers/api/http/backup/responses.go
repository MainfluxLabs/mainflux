// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package backup

import (
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

var (
	_ apiutil.Response = (*restoreMessagesRes)(nil)
)

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
	return len(res.file) == 0
}
