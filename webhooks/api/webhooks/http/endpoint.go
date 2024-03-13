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
			Name:   req.Name,
			Format: req.Format,
			Url:    req.Url,
		}
		_, err := svc.CreateWebhook(ctx, req.Token, wh)
		if err != nil {
			return nil, err
		}

		res := webhookRes{
			Created: true,
		}
		return res, nil
	}
}

func listWebhooks(svc webhooks.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listWebhooksReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		webhooks, err := svc.ListWebhooks(ctx, req.token)
		if err != nil {
			return nil, err
		}

		return buildWebhooksResponse(webhooks), nil
	}
}

func buildWebhooksResponse(webhooks []webhooks.Webhook) webhooksRes {
	res := webhooksRes{Webhooks: []webhookResponse{}}
	for _, wh := range webhooks {
		webhook := webhookResponse{
			ID:     wh.ID,
			Name:   wh.Name,
			Format: wh.Format,
			Url:    wh.Url,
		}
		res.Webhooks = append(res.Webhooks, webhook)
	}
	return res
}
