// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"

	"github.com/MainfluxLabs/mainflux/mqtt"
	"github.com/go-kit/kit/endpoint"
)

func listSubscriptions(svc mqtt.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listSubscriptionsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		subs, err := svc.ListSubscriptions(ctx, req.chanID, req.token, req.key, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		res := listSubscriptionsRes{
			pageRes: pageRes{
				Total:  subs.Total,
				Offset: subs.Offset,
				Limit:  subs.Limit,
			},
			Subscriptions: []viewSubRes{},
		}

		for _, sub := range subs.Subscriptions {
			view := viewSubRes{
				Subtopic:  sub.Subtopic,
				ThingID:   sub.ThingID,
				ChannelID: sub.ChanID,
				ClientID:  sub.ClientID,
				Status:    sub.Status,
				CreatedAt: sub.CreatedAt,
			}
			res.Subscriptions = append(res.Subscriptions, view)
		}

		return res, nil
	}
}
