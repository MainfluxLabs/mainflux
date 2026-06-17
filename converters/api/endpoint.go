// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"

	"github.com/MainfluxLabs/mainflux/converters"
	"github.com/go-kit/kit/endpoint"
)

func convertCSVToJSONEndpoint(svc converters.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(convertCSVReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		// Publish async to avoid blocking past the gateway timeout on large files.
		go func() { _ = svc.PublishJSONMessages(context.Background(), req.key.Value, req.csvLines) }()

		return nil, nil
	}
}

func convertCSVToSenMLEndpoint(svc converters.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(convertCSVReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		// Publish async to avoid blocking past the gateway timeout on large files.
		go func() { _ = svc.PublishSenMLMessages(context.Background(), req.key.Value, req.csvLines) }()

		return nil, nil
	}
}
