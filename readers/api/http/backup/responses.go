// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package backup

import (
	"encoding/json"
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

var (
	_ apiutil.Response = (*restoreRes)(nil)
)

type restoreRes struct{}

func (res restoreRes) Code() int {
	return http.StatusCreated
}

func (res restoreRes) Headers() map[string]string {
	return map[string]string{}
}

func (res restoreRes) Empty() bool {
	return true
}

type backupRes struct {
	JSONMessages  json.RawMessage `json:"json_messages"`
	SenMLMessages json.RawMessage `json:"senml_messages"`
}

func (res backupRes) Code() int {
	return http.StatusOK
}

func (res backupRes) Headers() map[string]string {
	return map[string]string{}
}

func (res backupRes) Empty() bool {
	return len(res.JSONMessages) == 0 && len(res.SenMLMessages) == 0
}
