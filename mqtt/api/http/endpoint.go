// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"

	"github.com/MainfluxLabs/mainflux/mqtt"
	"github.com/go-kit/kit/endpoint"
)

func listSubscriptionsEndpoint(svc mqtt.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listSubscriptionsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		subs, err := svc.ListSubscriptions(ctx, req.groupID, req.token, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		res := listSubscriptionsRes{
			pageRes: pageRes{
				Total:  subs.Total,
				Offset: req.pageMetadata.Offset,
				Limit:  req.pageMetadata.Limit,
			},
			Subscriptions: []viewSubRes{},
		}

		for _, sub := range subs.Subscriptions {
			view := viewSubRes{
				Subtopic:  sub.Subtopic,
				ThingID:   sub.ThingID,
				GroupID:   sub.GroupID,
				ClientID:  sub.ClientID,
				Status:    sub.Status,
				CreatedAt: sub.CreatedAt,
			}
			res.Subscriptions = append(res.Subscriptions, view)
		}

		return res, nil
	}
}
