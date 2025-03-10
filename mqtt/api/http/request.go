// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"github.com/MainfluxLabs/mainflux/mqtt"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

const maxLimitSize = 100

type apiReq interface {
	validate() error
}

type listSubscriptionsReq struct {
	groupID      string
	token        string
	key          string
	pageMetadata mqtt.PageMetadata
}

func (req listSubscriptionsReq) validate() error {
	if req.groupID == "" {
		return apiutil.ErrMissingGroupID
	}

	if req.token == "" && req.key == "" {
		return apiutil.ErrBearerToken
	}

	if req.pageMetadata.Limit > maxLimitSize {
		return apiutil.ErrLimitSize
	}

	return nil
}
