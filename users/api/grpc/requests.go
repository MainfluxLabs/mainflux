// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import "github.com/MainfluxLabs/mainflux/internal/apiutil"

type getUsersByIDsReq struct {
	IDs []string
}

func (req getUsersByIDsReq) validate() error {
	if len(req.IDs) == 0 {
		return apiutil.ErrMissingID
	}

	return nil
}
