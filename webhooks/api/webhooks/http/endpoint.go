// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"

	"github.com/MainfluxLabs/mainflux/webhooks"
	"github.com/go-kit/kit/endpoint"
)

func createWebhookEndpoint(svc webhooks.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(webhookReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		response, err := svc.CreateWebhook(req.name)
		if err != nil {
			return nil, err
		}

		res := webhookRes{
			created: response,
		}
		return res, nil
	}
}
