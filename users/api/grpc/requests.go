// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import "github.com/MainfluxLabs/mainflux/internal/apiutil"

type getUsersByIDsReq struct {
	ids []string
}

func (req getUsersByIDsReq) validate() error {
	if len(req.ids) == 0 {
		return apiutil.ErrMissingID
	}

	return nil
}
