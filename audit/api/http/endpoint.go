// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"

	"github.com/MainfluxLabs/mainflux/audit"
	"github.com/go-kit/kit/endpoint"
)

func listEventsEndpoint(svc audit.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listEventsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListEvents(ctx, req.token, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildEventsResponse(page, req.pageMetadata), nil
	}
}

func listEventsByOrgEndpoint(svc audit.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listEventsByOrgReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListEventsByOrg(ctx, req.token, req.orgID, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildEventsResponse(page, req.pageMetadata), nil
	}
}

func listEventsByGroupEndpoint(svc audit.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listEventsByGroupReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListEventsByGroup(ctx, req.token, req.groupID, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildEventsResponse(page, req.pageMetadata), nil
	}
}

func buildEventsResponse(ep audit.EventsPage, pm audit.PageMetadata) eventsPageRes {
	res := eventsPageRes{
		pageRes: pageRes{
			Total:  ep.Total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
			Order:  pm.Order,
			Dir:    pm.Dir,
		},
		Events: []eventRes{},
	}

	for _, e := range ep.Events {
		res.Events = append(res.Events, eventRes{
			ID:         e.ID,
			OccurredAt: e.OccurredAt,
			Operation:  e.Operation,
			Actor:      e.Actor,
			OrgID:      e.OrgID,
			GroupID:    e.GroupID,
			Data:       e.Data,
		})
	}

	return res
}
