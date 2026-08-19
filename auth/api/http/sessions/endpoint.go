// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sessions

import (
	"context"
	"net/http"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/go-kit/kit/endpoint"
)

type sessionReq struct {
	token string
}

func (req sessionReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	return nil
}

type logoutRes struct{}

func (res logoutRes) Code() int { return http.StatusNoContent }

func (res logoutRes) Headers() map[string]string { return map[string]string{} }

func (res logoutRes) Empty() bool { return true }

// logoutEndpoint ends the session the presented token belongs to, leaving the
// user's other sessions running.
func logoutEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(sessionReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.Logout(ctx, req.token); err != nil {
			return nil, err
		}

		return logoutRes{}, nil
	}
}

// logoutAllEndpoint ends every session belonging to the token's owner.
func logoutAllEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(sessionReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.LogoutAll(ctx, req.token); err != nil {
			return nil, err
		}

		return logoutRes{}, nil
	}
}
