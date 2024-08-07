// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import "github.com/MainfluxLabs/mainflux/pkg/apiutil"

type getUsersByIDsReq struct {
	ids []string
}

func (req getUsersByIDsReq) validate() error {
	if len(req.ids) == 0 {
		return apiutil.ErrMissingID
	}

	return nil
}

type getUsersByEmailsReq struct {
	emails []string
}

func (req getUsersByEmailsReq) validate() error {
	if len(req.emails) == 0 {
		return apiutil.ErrMissingEmail
	}

	return nil
}
