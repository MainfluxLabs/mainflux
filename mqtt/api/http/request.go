// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"github.com/MainfluxLabs/mainflux/internal/apiutil"
	"github.com/MainfluxLabs/mainflux/mqtt"
)

const maxLimitSize = 100

type apiReq interface {
	validate() error
}

type listSubscriptionsReq struct {
	chanID       string
	token        string
	pageMetadata mqtt.PageMetadata
}

func (req listSubscriptionsReq) validate() error {
	if req.chanID == "" {
		return apiutil.ErrMissingID
	}

	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.pageMetadata.Limit > maxLimitSize {
		return apiutil.ErrLimitSize
	}

	return nil
}
