// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package backup

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

type backupMessagesReq struct {
	token string
}

func (req backupMessagesReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	return nil
}

type restoreMessagesReq struct {
	token    string
	Messages []byte
}

func (req restoreMessagesReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.Messages) == 0 {
		return apiutil.ErrEmptyList
	}

	return nil
}
