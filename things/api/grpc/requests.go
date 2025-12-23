// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/things"
)

type thingKey struct {
	value   string
	keyType string
}

func (req thingKey) validate() error {
	if req.keyType != things.KeyTypeInternal && req.keyType != things.KeyTypeExternal {
		return apiutil.ErrInvalidThingKeyType
	}

	if req.value == "" {
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
	thingKey
	id string
}

func (req thingAccessGroupReq) validate() error {
	if err := req.thingKey.validate(); err != nil {
		return err
	}

	if req.id == "" {
		return apiutil.ErrMissingGroupID
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

type groupMembership struct {
	userID  string
	groupID string
	role    string
}

type createGroupMembershipsReq struct {
	memberships []groupMembership
}

func (req createGroupMembershipsReq) validate() error {
	for _, memb := range req.memberships {
		if memb.userID == "" {
			return apiutil.ErrMissingUserID
		}

		if memb.groupID == "" {
			return apiutil.ErrMissingGroupID
		}

		if memb.role == "" {
			return apiutil.ErrMissingRole
		}
	}

	return nil
}

type getGroupReq struct {
	groupID string
}

func (req getGroupReq) validate() error {
	if req.groupID == "" {
		return apiutil.ErrMissingGroupID
	}

	return nil
}
