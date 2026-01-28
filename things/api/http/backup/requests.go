// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package backup

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/things/api/http/memberships"
)

type backupReq struct {
	token string
}

func (req backupReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	return nil
}

type restoreReq struct {
	token            string
	Things           []viewThingRes                       `json:"things"`
	Profiles         []backupProfile                      `json:"profiles"`
	Groups           []backupGroup                        `json:"groups"`
	GroupMemberships []memberships.ViewGroupMembershipRes `json:"group_memberships"`
}

func (req restoreReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if len(req.Things) == 0 {
		return apiutil.ErrEmptyList
	}

	return nil
}
