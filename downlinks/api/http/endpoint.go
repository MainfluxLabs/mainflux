// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"

	"github.com/MainfluxLabs/mainflux/downlinks"
	"github.com/MainfluxLabs/mainflux/pkg/cron"
	"github.com/go-kit/kit/endpoint"
)

func createDownlinksEndpoint(svc downlinks.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(createDownlinksReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		var dls []downlinks.Downlink
		for _, dReq := range req.Downlinks {
			scheduler := cron.NormalizeTimezone(dReq.Scheduler)

			dl := downlinks.Downlink{
				Name:       dReq.Name,
				Url:        dReq.Url,
				Method:     dReq.Method,
				Payload:    []byte(dReq.Payload),
				Headers:    dReq.Headers,
				Scheduler:  scheduler,
				TimeFilter: dReq.TimeFilter,
				Metadata:   dReq.Metadata,
			}
			dls = append(dls, dl)

		}

		saved, err := svc.CreateDownlinks(ctx, req.token, req.thingID, dls...)
		if err != nil {
			return nil, err
		}

		return buildDownlinksResponse(saved, true), nil
	}
}

func listDownlinksByThingEndpoint(svc downlinks.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listThingDownlinksReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		dls, err := svc.ListDownlinksByThing(ctx, req.token, req.thingID, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildDownlinksPageResponse(dls), nil
	}
}

func listDownlinksByGroupEndpoint(svc downlinks.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listDownlinksReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		dls, err := svc.ListDownlinksByGroup(ctx, req.token, req.groupID, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildDownlinksPageResponse(dls), nil
	}
}

func viewDownlinkEndpoint(svc downlinks.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(downlinkReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		downlink, err := svc.ViewDownlink(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		res := buildDownlinkResponse(downlink)
		res.updated = false

		return res, nil
	}
}

func updateDownlinkEndpoint(svc downlinks.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(updateDownlinkReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		scheduler := cron.NormalizeTimezone(req.Scheduler)

		downlink := downlinks.Downlink{
			ID:         req.id,
			Name:       req.Name,
			Url:        req.Url,
			Method:     req.Method,
			Payload:    []byte(req.Payload),
			Headers:    req.Headers,
			Scheduler:  scheduler,
			TimeFilter: req.TimeFilter,
			Metadata:   req.Metadata,
		}

		if err := svc.UpdateDownlink(ctx, req.token, downlink); err != nil {
			return nil, err
		}

		return downlinkResponse{updated: true}, nil
	}
}

func removeDownlinksEndpoint(svc downlinks.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(removeDownlinksReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RemoveDownlinks(ctx, req.token, req.DownlinkIDs...); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func buildDownlinksResponse(dls []downlinks.Downlink, created bool) downlinksRes {
	res := downlinksRes{Downlinks: []downlinkResponse{}, created: created}
	for _, dl := range dls {
		res.Downlinks = append(res.Downlinks, buildDownlinkResponse(dl))
	}

	return res
}

func buildDownlinkResponse(downlink downlinks.Downlink) downlinkResponse {
	dl := downlinkResponse{
		ID:         downlink.ID,
		GroupID:    downlink.GroupID,
		ThingID:    downlink.ThingID,
		Name:       downlink.Name,
		Url:        downlink.Url,
		Method:     downlink.Method,
		Payload:    string(downlink.Payload),
		ResHeaders: downlink.Headers,
		Scheduler:  downlink.Scheduler,
		TimeFilter: downlink.TimeFilter,
		Metadata:   downlink.Metadata,
	}

	return dl
}

func buildDownlinksPageResponse(dp downlinks.DownlinksPage) downlinksPageRes {
	res := downlinksPageRes{
		pageRes: pageRes{
			Total:  dp.Total,
			Offset: dp.Offset,
			Limit:  dp.Limit,
		},
		Downlinks: []downlinkResponse{},
	}

	for _, downlink := range dp.Downlinks {
		res.Downlinks = append(res.Downlinks, buildDownlinkResponse(downlink))
	}

	return res
}
