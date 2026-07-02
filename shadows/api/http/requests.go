// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/shadows"
)

type updateDesiredStateReq struct {
	token   string
	thingID string
	Desired shadows.State `json:"desired"`
}

func (req updateDesiredStateReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.thingID == "" {
		return apiutil.ErrMissingThingID
	}

	if len(req.Desired) == 0 {
		return apiutil.ErrEmptyList
	}

	return nil
}

type shadowReq struct {
	token   string
	thingID string
}

func (req shadowReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.thingID == "" {
		return apiutil.ErrMissingThingID
	}

	return nil
}
