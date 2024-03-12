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

		wh := webhooks.Webhook{
			Name:   req.name,
			Format: req.format,
			Url:    req.url,
		}
		_, err := svc.CreateWebhook(ctx, req.token, wh)
		if err != nil {
			return nil, err
		}

		res := webhookRes{
			created: true,
		}
		return res, nil
	}
}
