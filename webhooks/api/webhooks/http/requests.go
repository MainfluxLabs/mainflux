// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import "github.com/MainfluxLabs/mainflux/webhooks"

type apiReq interface {
	validate() error
}

type webhookReq struct {
	name   string `json:"name"`
	format string `json:"format"`
	url    string `json:"url"`
	token  string `json:"token"`
}

func (req webhookReq) validate() error {
	if req.name == "" {
		return webhooks.ErrMalformedEntity
	}

	return nil
}
