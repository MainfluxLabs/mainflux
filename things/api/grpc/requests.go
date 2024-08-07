// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import "github.com/MainfluxLabs/mainflux/pkg/apiutil"

type connByKeyReq struct {
	key string
}

func (req connByKeyReq) validate() error {
	if req.key == "" {
		return apiutil.ErrBearerKey
	}

	return nil
}

type accessByIDReq struct {
	thingID string
	chanID  string
}

func (req accessByIDReq) validate() error {
	if req.thingID == "" || req.chanID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type channelOwnerReq struct {
	token  string
	chanID string
}

func (req channelOwnerReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.chanID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type accessGroupReq struct {
	token   string
	groupID string
	action  string
}

func (req accessGroupReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
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
