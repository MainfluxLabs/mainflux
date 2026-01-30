// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/things"
)

type publishReq struct {
	msg protomfx.Message
	things.ThingKey
}

func (req publishReq) validate() error {
	if err := req.ThingKey.Validate(); err != nil {
		return err
	}

	return nil
}

type cmdReq struct {
	token string
	id    string
	msg   protomfx.Message
}

type thingCommandReq struct {
	cmdReq
}

func (req thingCommandReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingThingID
	}

	return nil
}

type groupCommandReq struct {
	cmdReq
}

func (req groupCommandReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingGroupID
	}

	return nil
}
