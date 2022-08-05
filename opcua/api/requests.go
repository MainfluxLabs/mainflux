// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import "github.com/MainfluxLabs/mainflux/internal/apiutil"

type browseReq struct {
	ServerURI  string
	Namespace  string
	Identifier string
}

func (req *browseReq) validate() error {
	if req.ServerURI == "" {
		return apiutil.ErrMissingID
	}

	return nil
}
