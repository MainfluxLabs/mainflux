// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
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

type authReq struct {
	Token   string
	Object  string
	Subject string
	Action  string
}

func (req authReq) validate() error {
	if req.Token == "" {
		return apiutil.ErrBearerToken
	}

	if req.Subject != auth.RootSubject &&
		req.Subject != auth.OrgsSubject {
		return apiutil.ErrInvalidSubject
	}

	return nil
}

type assignRoleReq struct {
	ID   string
	Role string
}

func (req assignRoleReq) validate() error {
	if req.Role == "" {
		return apiutil.ErrMissingRole
	}

	if req.ID == "" {
		return apiutil.ErrMissingID
	}

	return nil
}

type retrieveRoleReq struct {
	id string
}

func (req retrieveRoleReq) validate() error {
	if req.id == "" {
		return apiutil.ErrMissingID
	}

	return nil
}
