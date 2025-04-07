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
		return apiutil.ErrMissingThingID
	}

	return nil
}

type accessReq struct {
	token  string
	action string
}

func (req accessReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.action != things.Admin && req.action != things.Viewer && req.action != things.Editor {
		return apiutil.ErrInvalidAction
	}

	return nil
}

type userAccessThingReq struct {
	accessReq
	id string
}

func (req userAccessThingReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingThingID
	}

	return req.accessReq.validate()
}

type userAccessProfileReq struct {
	accessReq
	id string
}

func (req userAccessProfileReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingProfileID
	}

	return req.accessReq.validate()
}

type userAccessGroupReq struct {
	accessReq
	id string
}

func (req userAccessGroupReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingGroupID
	}

	return req.accessReq.validate()
}

type thingAccessGroupReq struct {
	key string
	id  string
}

func (req thingAccessGroupReq) validate() error {
	if req.key == "" {
		return apiutil.ErrBearerKey
	}

	if req.id == "" {
		return apiutil.ErrMissingGroupID
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

type groupIDByThingIDReq struct {
	thingID string
}

func (req groupIDByThingIDReq) validate() error {
	if req.thingID == "" {
		return apiutil.ErrMissingThingID
	}

	return nil
}
