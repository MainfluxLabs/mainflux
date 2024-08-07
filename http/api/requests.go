// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

type publishReq struct {
	msg   protomfx.Message
	token string
}

func (req publishReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	return nil
}
