// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sessions

import (
	"context"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/go-kit/kit/endpoint"
)

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
