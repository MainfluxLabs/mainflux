// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"

	"github.com/MainfluxLabs/mainflux/things"
	"github.com/go-kit/kit/endpoint"
)

func identifyEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(identifyReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		id, err := svc.Identify(ctx, req.Token)
		if err != nil {
			return nil, err
		}

		res := identityRes{
			ID: id,
		}

		return res, nil
	}
}

func getConnByKeyEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getConnByKeyReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		conn, err := svc.GetConnByKey(ctx, req.Token)
		if err != nil {
			return nil, err
		}

		res := connByKeyRes{
			ChannelID: conn.ChannelID,
			ThingID:   conn.ThingID,
		}

		return res, nil
	}
}
