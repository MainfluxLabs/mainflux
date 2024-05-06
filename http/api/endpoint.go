// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"

	"github.com/MainfluxLabs/mainflux/http"
	"github.com/go-kit/kit/endpoint"
)

func sendMessageEndpoint(svc http.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(publishReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		_, err := svc.Publish(ctx, req.token, req.msg)
		return nil, err
	}
}
