// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/go-kit/kit/endpoint"
)

func listJSONMessagesEndpoint(svc readers.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listJSONMessagesReq)
		page, err := svc.ListJSONMessages(ctx, "", req.thingKey, req.pm)
		return listJSONMessagesRes{page: page}, err
	}
}

func listSenMLMessagesEndpoint(svc readers.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listSenMLMessagesReq)
		page, err := svc.ListSenMLMessages(ctx, "", req.thingKey, req.pm)
		return listSenMLMessagesRes{page: page}, err
	}
}
