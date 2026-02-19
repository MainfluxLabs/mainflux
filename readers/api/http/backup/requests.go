// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package backup

import (
	"encoding/json"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

type backupReq struct {
	token string
}

func (req backupReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	return nil
}

type restoreReq struct {
	token         string
	JSONMessages  json.RawMessage `json:"json_messages"`
	SenMLMessages json.RawMessage `json:"senml_messages"`
}

func (req restoreReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.JSONMessages) == 0 && len(req.SenMLMessages) == 0 {
		return apiutil.ErrEmptyList
	}

	return nil
}
