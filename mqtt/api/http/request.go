// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"github.com/MainfluxLabs/mainflux/mqtt"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

const maxLimitSize = 100

var errAuthHeader = errors.New("missing or invalid auth header")

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
		return errAuthHeader
	}

	if req.pageMetadata.Limit > maxLimitSize {
		return apiutil.ErrLimitSize
	}

	return nil
}
