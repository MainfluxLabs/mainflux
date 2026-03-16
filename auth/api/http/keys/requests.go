// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package keys

import (
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

const (
	maxLimitSize = 200
)

type issueKeyReq struct {
	token    string
	Type     uint32        `json:"type,omitempty"`
	Duration time.Duration `json:"duration,omitempty"`
}

// It is not possible to issue Reset key using HTTP API.
func (req issueKeyReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.Type != auth.LoginKey &&
		req.Type != auth.RecoveryKey &&
		req.Type != auth.APIKey {
		return apiutil.ErrInvalidAPIKey
	}

	return nil
}

type keyReq struct {
	token string
	id    string
}

func (req keyReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingKeyID
	}
	return nil
}

type listKeysReq struct {
	token        string
	pageMetadata auth.PageMetadata
}

func (req listKeysReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.pageMetadata.Limit > maxLimitSize {
		return apiutil.ErrLimitSize
	}

	if req.pageMetadata.Dir != "" &&
		req.pageMetadata.Dir != apiutil.AscDir && req.pageMetadata.Dir != apiutil.DescDir {
		return apiutil.ErrInvalidDirection
	}

	return nil
}
