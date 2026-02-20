// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package things

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

const (
	maxLimitSize = 200
	maxNameSize  = 1024
)

var ErrMissingConfig = errors.New("missing config")

type viewThingConfigReq struct {
	token   string
	thingID string
}

func (req *viewThingConfigReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.thingID == "" {
		return apiutil.ErrMissingThingID
	}

	return nil
}

type listThingsConfigsReq struct {
	token        string
	pageMetadata apiutil.PageMetadata
}

func (req *listThingsConfigsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	return apiutil.ValidatePageMetadata(req.pageMetadata, maxLimitSize, maxNameSize)
}

type updateThingConfigReq struct {
	token   string
	thingID string
	Config  map[string]any `json:"config"`
}

func (req updateThingConfigReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.thingID == "" {
		return apiutil.ErrMissingThingID
	}

	return nil
}
