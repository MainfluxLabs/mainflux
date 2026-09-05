// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sessions

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

type sessionReq struct {
	token string
}

func (req sessionReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	return nil
}
