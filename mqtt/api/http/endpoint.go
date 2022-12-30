// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"

	"github.com/MainfluxLabs/mainflux/mqtt"
	"github.com/go-kit/kit/endpoint"
)

func listAllSubscriptions(svc mqtt.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listAllSubscriptionsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		subs, err := svc.ListSubscriptions(ctx, req.token, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		res := listAllSubscriptionsRes{
			pageRes: pageRes{
				Total:  subs.Total,
				Offset: subs.Offset,
				Limit:  subs.Limit,
			},
			Subscriptions: []viewSubRes{},
		}
		for _, sub := range subs.Subscriptions {
			view := viewSubRes{
                OwnerID:   sub.OwnerID,
				Subtopic:  sub.Subtopic,
				ThingID:   sub.ThingID,
				ChannelID: sub.ChanID,
			}
			res.Subscriptions = append(res.Subscriptions, view)
		}

		return res, nil
	}
}
