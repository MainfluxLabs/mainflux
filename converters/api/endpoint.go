// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"

	"github.com/MainfluxLabs/mainflux/converters"
	"github.com/go-kit/kit/endpoint"
)

func convertCSVEndpoint(svc converters.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(convertCSVReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		switch req.to {
		case "senml":
			// Publish async to avoid blocking past the gateway timeout on large files.
			go func() { _ = svc.PublishSenMLMessagesFromCSV(context.Background(), req.key.Value, req.csvLines) }()
		default:
			go func() { _ = svc.PublishJSONMessagesFromCSV(context.Background(), req.key.Value, req.csvLines) }()
		}

		return nil, nil
	}
}

func convertJSONEndpoint(svc converters.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(convertJSONReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		switch req.to {
		case "senml":
			// Publish async to avoid blocking past the gateway timeout on large files.
			go func() { _ = svc.PublishSenMLMessagesFromJSON(context.Background(), req.key.Value, req.records) }()
		default:
			go func() { _ = svc.PublishJSONMessagesFromJSON(context.Background(), req.key.Value, req.records) }()
		}

		return nil, nil
	}
}
