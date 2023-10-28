// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import "github.com/MainfluxLabs/mainflux/internal/apiutil"

type identifyReq struct {
	Token string `json:"token"`
}

func (req identifyReq) validate() error {
	if req.Token == "" {
		return apiutil.ErrBearerKey
	}

	return nil
}

type getConnByKeyReq struct {
	chanID string
	Key    string `json:"key"`
}

func (req getConnByKeyReq) validate() error {
	if req.Key == "" {
		return apiutil.ErrBearerKey
	}

	return nil
}
