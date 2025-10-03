// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"github.com/MainfluxLabs/mainflux/mqtt"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

const maxLimitSize = 200

type apiReq interface {
	validate() error
}

type listSubscriptionsReq struct {
	groupID      string
	token        string
	thingKey     apiutil.ThingKey
	pageMetadata mqtt.PageMetadata
}

func (req listSubscriptionsReq) validate() error {
	if req.groupID == "" {
		return apiutil.ErrMissingGroupID
	}

	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if err := req.thingKey.Validate(); err != nil {
		return err
	}

	if req.pageMetadata.Limit > maxLimitSize {
		return apiutil.ErrLimitSize
	}

	return nil
}
