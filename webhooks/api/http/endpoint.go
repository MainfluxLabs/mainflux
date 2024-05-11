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
				GroupID: req.groupID,
				Name:    wReq.Name,
				Url:     wReq.Url,
				Headers: wReq.Headers,
			}
			whs = append(whs, wh)
		}

		saved, err := svc.CreateWebhooks(ctx, req.token, whs...)
		if err != nil {
			return nil, err
		}

		return buildWebhooksResponse(saved, true), nil
	}
}

func listWebhooksByGroupEndpoint(svc webhooks.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(webhookReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		whs, err := svc.ListWebhooksByGroup(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		return buildWebhooksResponse(whs, false), nil
	}
}

func buildWebhooksResponse(webhooks []webhooks.Webhook, created bool) webhooksRes {
	res := webhooksRes{Webhooks: []webhookResponse{}, created: created}
	for _, wh := range webhooks {
		webhook := webhookResponse{
			ID:             wh.ID,
			GroupID:        wh.GroupID,
			Name:           wh.Name,
			Url:            wh.Url,
			WebhookHeaders: wh.Headers,
		}
		res.Webhooks = append(res.Webhooks, webhook)
	}

	return res
}
