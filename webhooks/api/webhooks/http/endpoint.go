// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"github.com/MainfluxLabs/mainflux/webhooks"
	"github.com/go-kit/kit/endpoint"
)

func createWebhooksEndpoint(svc webhooks.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createWebhooksReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		whs := []webhooks.Webhook{}
		for _, wReq := range req.Webhooks {
			wh := webhooks.Webhook{
				ThingID: req.ThingID,
				Name:    wReq.Name,
				Format:  wReq.Format,
				Url:     wReq.Url,
			}
			whs = append(whs, wh)
		}

		_, err := svc.CreateWebhooks(ctx, req.Token, whs...)
		if err != nil {
			return nil, err
		}

		res := webhookRes{
			Created: true,
		}
		return res, nil
	}
}

func listWebhooksByThing(svc webhooks.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listWebhooksReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		webhooks, err := svc.ListWebhooksByThing(ctx, req.Token, req.ThingID)
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
			ThingID: wh.ThingID,
			Name:    wh.Name,
			Format:  wh.Format,
			Url:     wh.Url,
		}
		res.Webhooks = append(res.Webhooks, webhook)
	}

	return res
}
