// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/things"
)

type thingKey struct {
	key     string
	keyType string
}

func (req thingKey) validate() error {
	if req.keyType != things.KeyTypeInternal && req.keyType != things.KeyTypeExternal {
		return apiutil.ErrInvalidThingKeyType
	}

	if req.key == "" {
		return apiutil.ErrBearerKey
	}

	return nil
}

type thingIDReq struct {
	thingID string
}

func (req thingIDReq) validate() error {
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
	key     string
	keyType string
	id      string
}

func (req thingAccessGroupReq) validate() error {
	if req.key == "" {
		return apiutil.ErrBearerKey
	}

	if req.keyType != things.KeyTypeInternal && req.keyType != things.KeyTypeExternal {
		return apiutil.ErrInvalidThingKeyType
	}

	if req.id == "" {
		return apiutil.ErrMissingGroupID
	}

	return nil
}

type identifyReq struct {
	key     string
	keyType string
}

func (req identifyReq) validate() error {
	if req.key == "" {
		return apiutil.ErrBearerKey
	}

	if req.keyType != things.KeyTypeInternal && req.keyType != things.KeyTypeExternal {
		return apiutil.ErrInvalidThingKeyType
	}

	return nil
}

type profileIDReq struct {
	profileID string
}

func (req profileIDReq) validate() error {
	if req.profileID == "" {
		return apiutil.ErrMissingProfileID
	}

	return nil
}

type orgAccessReq struct {
	orgID string
	token string
}

func (req orgAccessReq) validate() error {
	if req.orgID == "" {
		return apiutil.ErrMissingOrgID
	}
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	return nil
}
