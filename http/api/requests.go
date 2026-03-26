// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

type publishReq struct {
	msg protomfx.Message
	domain.ThingKey
}

func (req publishReq) validate() error {
	if err := apiutil.ValidateThingKey(req.ThingKey); err != nil {
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
