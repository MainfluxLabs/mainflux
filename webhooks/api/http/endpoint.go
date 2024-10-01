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
				GroupID:  req.groupID,
				Name:     wReq.Name,
				Url:      wReq.Url,
				Headers:  wReq.Headers,
				Metadata: wReq.Metadata,
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
		req := request.(listWebhooksReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListWebhooksByGroup(ctx, req.token, req.id, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildWebhooksByGroupResponse(page), nil
	}
}

func viewWebhookEndpoint(svc webhooks.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(webhookReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		webhook, err := svc.ViewWebhook(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		return buildWebhookResponse(webhook, false), nil
	}
}

func updateWebhookEndpoint(svc webhooks.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateWebhookReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		webhook := webhooks.Webhook{
			ID:       req.id,
			Name:     req.Name,
			Url:      req.Url,
			Headers:  req.Headers,
			Metadata: req.Metadata,
		}

		if err := svc.UpdateWebhook(ctx, req.token, webhook); err != nil {
			return nil, err
		}

		return webhookResponse{updated: true}, nil
	}
}

func removeWebhooksEndpoint(svc webhooks.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(removeWebhooksReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RemoveWebhooks(ctx, req.token, req.WebhookIDs...); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func buildWebhooksByGroupResponse(wp webhooks.WebhooksPage) WebhooksPageRes {
	res := WebhooksPageRes{
		pageRes: pageRes{
			Total:  wp.Total,
			Offset: wp.Offset,
			Limit:  wp.Limit,
		},
		Webhooks: []webhookResponse{},
	}

	for _, wh := range wp.Webhooks {
		webhook := webhookResponse{
			ID:         wh.ID,
			GroupID:    wh.GroupID,
			Name:       wh.Name,
			Url:        wh.Url,
			ResHeaders: wh.Headers,
			Metadata:   wh.Metadata,
		}
		res.Webhooks = append(res.Webhooks, webhook)
	}

	return res
}

func buildWebhooksResponse(webhooks []webhooks.Webhook, created bool) webhooksRes {
	res := webhooksRes{Webhooks: []webhookResponse{}, created: created}
	for _, wh := range webhooks {
		webhook := webhookResponse{
			ID:         wh.ID,
			GroupID:    wh.GroupID,
			Name:       wh.Name,
			Url:        wh.Url,
			ResHeaders: wh.Headers,
			Metadata:   wh.Metadata,
		}
		res.Webhooks = append(res.Webhooks, webhook)
	}

	return res
}

func buildWebhookResponse(webhook webhooks.Webhook, updated bool) webhookResponse {
	wh := webhookResponse{
		ID:         webhook.ID,
		GroupID:    webhook.GroupID,
		Name:       webhook.Name,
		Url:        webhook.Url,
		ResHeaders: webhook.Headers,
		Metadata:   webhook.Metadata,
		updated:    updated,
	}

	return wh
}
