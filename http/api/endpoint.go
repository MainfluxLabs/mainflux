// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"

	"github.com/MainfluxLabs/mainflux/http"
	"github.com/go-kit/kit/endpoint"
)

func sendMessageEndpoint(svc http.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(publishReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		return nil, svc.Publish(ctx, req.ThingKey, req.msg)
	}
}

func sendCommandToThingEndpoint(svc http.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(thingCommandReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		return nil, svc.SendCommandToThing(ctx, req.token, req.id, req.msg)
	}
}

func sendCommandToGroupEndpoint(svc http.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(groupCommandReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		return nil, svc.SendCommandToGroup(ctx, req.token, req.id, req.msg)
	}
}
