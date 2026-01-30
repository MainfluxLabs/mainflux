// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package backup

import (
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
	token    string
	Messages []byte
}

func (req restoreReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.Messages) == 0 {
		return apiutil.ErrEmptyList
	}

	return nil
}
