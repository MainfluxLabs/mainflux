// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/internal/apiutil"
)

type identityReq struct {
	token string
	kind  uint32
}

func (req identityReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.kind != auth.LoginKey &&
		req.kind != auth.APIKey &&
		req.kind != auth.RecoveryKey {
		return apiutil.ErrInvalidAuthKey
	}

	return nil
}

type issueReq struct {
	id      string
	email   string
	keyType uint32
}

func (req issueReq) validate() error {
	if req.email == "" {
		return apiutil.ErrMissingEmail
	}
	if req.keyType != auth.LoginKey &&
		req.keyType != auth.APIKey &&
		req.keyType != auth.RecoveryKey {
		return apiutil.ErrInvalidAuthKey
	}

	return nil
}

type assignReq struct {
	token    string
	groupID  string
	memberID string
}

func (req assignReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.groupID == "" || req.memberID == "" {
		return apiutil.ErrMissingID
	}
	return nil
}

type membersReq struct {
	token      string
	groupID    string
	offset     uint64
	limit      uint64
	memberType string
}

func (req membersReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.groupID == "" {
		return apiutil.ErrMissingID
	}
	if req.memberType == "" {
		return apiutil.ErrMissingMemberType
	}
	return nil
}

type authReq struct {
	Email string
}

func (req authReq) validate() error {
	if req.Email == "" {
		return apiutil.ErrMissingEmail
	}

	return nil
}

type accessGroupReq struct {
	Token   string
	GroupID string
}

func (req accessGroupReq) validate() error {
	if req.Token == "" {
		return apiutil.ErrBearerToken
	}

	if req.GroupID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}
