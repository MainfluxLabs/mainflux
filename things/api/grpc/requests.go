// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/things"
)

type connByKeyReq struct {
	key string
}

func (req connByKeyReq) validate() error {
	if req.key == "" {
		return apiutil.ErrBearerKey
	}

	return nil
}

type authorizeReq struct {
	token   string
	object  string
	subject string
	action  string
}

func (req authorizeReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.object == "" {
		return apiutil.ErrMissingID
	}

	if req.subject != things.ThingSub && req.subject != things.ProfileSub && req.subject != things.GroupSub {
		return apiutil.ErrInvalidSubject
	}

	if req.action != things.Admin && req.action != things.Viewer && req.action != things.Editor {
		return apiutil.ErrInvalidAction
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

type configByThingIDReq struct {
	thingID string
}

func (req configByThingIDReq) validate() error {
	if req.thingID == "" {
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
