// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"

	"github.com/MainfluxLabs/mainflux/shadows"
	"github.com/go-kit/kit/endpoint"
)

func updateDesiredStateEndpoint(svc shadows.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(updateDesiredStateReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		sh, err := svc.UpdateDesiredState(ctx, req.token, req.thingID, req.Desired)
		if err != nil {
			return nil, err
		}

		return buildShadowResponse(sh), nil
	}
}

func viewShadowEndpoint(svc shadows.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(shadowReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		sh, err := svc.ViewShadow(ctx, req.token, req.thingID)
		if err != nil {
			return nil, err
		}

		return buildShadowResponse(sh), nil
	}
}

func removeShadowEndpoint(svc shadows.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(shadowReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RemoveShadow(ctx, req.token, req.thingID); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}
