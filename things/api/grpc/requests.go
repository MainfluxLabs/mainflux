// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import "github.com/MainfluxLabs/mainflux/pkg/apiutil"

const (
	subThing   = "thing"
	subChannel = "channel"
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

type accessGroupReq struct {
	token   string
	groupID string
	action  string
	object  string
	subject string
}

func (req accessGroupReq) validate() error {
	if req.subject != subThing && req.subject != subChannel && req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingID
	}

	if (req.subject == subThing || req.subject == subChannel) && req.object == "" {
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

type profileByThingReq struct {
	thingID string
}

func (req profileByThingReq) validate() error {
	if req.thingID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}
