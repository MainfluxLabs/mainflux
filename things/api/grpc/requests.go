// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/things"
)

type pubConfByKeyReq struct {
	key string
}

func (req pubConfByKeyReq) validate() error {
	if req.key == "" {
		return apiutil.ErrBearerKey
	}

	return nil
}

type configByThingIDReq struct {
	thingID string
}

func (req configByThingIDReq) validate() error {
	if req.thingID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type userAccessReq struct {
	token  string
	id     string
	action string
}

func (req userAccessReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingID
	}

	if req.action != things.Admin && req.action != things.Viewer && req.action != things.Editor {
		return apiutil.ErrInvalidAction
	}

	return nil
}

type thingAccessReq struct {
	key string
	id  string
}

func (req thingAccessReq) validate() error {
	if req.key == "" {
		return apiutil.ErrBearerKey
	}

	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type identifyReq struct {
	key string
}

func (req identifyReq) validate() error {
	if req.key == "" {
		return apiutil.ErrBearerKey
	}

	return nil
}

type getGroupsByIDsReq struct {
	ids []string
}

func (req getGroupsByIDsReq) validate() error {
	if len(req.ids) == 0 {
		return apiutil.ErrMissingID
	}

	return nil
}

type groupIDByThingIDReq struct {
	thingID string
}

func (req groupIDByThingIDReq) validate() error {
	if req.thingID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}
