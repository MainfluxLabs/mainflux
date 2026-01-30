// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package things

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/go-kit/kit/endpoint"
)

func createThingsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(createThingsReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		ths := []things.Thing{}
		for _, t := range req.Things {
			th := things.Thing{
				ID:          t.ID,
				Name:        t.Name,
				Key:         t.Key,
				ExternalKey: t.ExternalKey,
				Metadata:    t.Metadata,
			}
			ths = append(ths, th)
		}

		saved, err := svc.CreateThings(ctx, req.token, req.profileID, ths...)
		if err != nil {
			return nil, err
		}

		res := thingsRes{
			Things:  []thingRes{},
			created: true,
		}

		for _, t := range saved {
			th := thingRes{
				ID:          t.ID,
				GroupID:     t.GroupID,
				ProfileID:   t.ProfileID,
				Name:        t.Name,
				Key:         t.Key,
				ExternalKey: t.ExternalKey,
				Metadata:    t.Metadata,
			}
			res.Things = append(res.Things, th)
		}

		return res, nil
	}
}

func viewThingEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(resourceReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		thing, err := svc.ViewThing(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		res := viewThingRes{
			ID:          thing.ID,
			GroupID:     thing.GroupID,
			ProfileID:   thing.ProfileID,
			Name:        thing.Name,
			Key:         thing.Key,
			ExternalKey: thing.ExternalKey,
			Metadata:    thing.Metadata,
		}
		return res, nil
	}
}

func viewMetadataByKeyEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(viewMetadataReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		metadata, err := svc.ViewMetadataByKey(ctx, req.ThingKey)
		if err != nil {
			return nil, err
		}

		res := viewMetadataRes{
			Metadata: metadata,
		}

		return res, nil
	}
}

func listThingsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListThings(ctx, req.token, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildThingsResponse(page, req.pageMetadata), nil
	}
}

func listThingsByProfileEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listByProfileReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListThingsByProfile(ctx, req.token, req.id, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildThingsResponse(page, req.pageMetadata), nil
	}
}

func listThingsByGroupEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listByGroupReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListThingsByGroup(ctx, req.token, req.id, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildThingsResponse(page, req.pageMetadata), nil
	}
}

func listThingsByOrgEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listByOrgReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListThingsByOrg(ctx, req.token, req.id, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildThingsResponse(page, req.pageMetadata), nil
	}
}

func updateThingEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(updateThingReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		thing := things.Thing{
			ID:       req.id,
			Name:     req.Name,
			Key:      req.Key,
			Metadata: req.Metadata,
		}

		if err := svc.UpdateThing(ctx, req.token, thing); err != nil {
			return nil, err
		}

		res := thingRes{ID: req.id, created: false}
		return res, nil
	}
}

func updateThingGroupAndProfileEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(updateThingGroupAndProfileReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		thing := things.Thing{
			ID:        req.id,
			ProfileID: req.ProfileID,
			GroupID:   req.GroupID,
		}

		if err := svc.UpdateThingGroupAndProfile(ctx, req.token, thing); err != nil {
			return nil, err
		}

		res := thingRes{ID: req.id, created: false}
		return res, nil
	}
}

func updateThingsMetadataEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(updateThingsMetadataReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		ths := []things.Thing{}
		for _, t := range req.Things {
			th := things.Thing{
				ID:       t.ID,
				Metadata: t.Metadata,
			}
			ths = append(ths, th)
		}

		if err := svc.UpdateThingsMetadata(ctx, req.token, ths...); err != nil {
			return nil, err
		}

		res := thingsRes{
			created: false,
		}
		return res, nil
	}
}

func removeThingEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(resourceReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RemoveThings(ctx, req.token, req.id); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func removeThingsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(removeThingsReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RemoveThings(ctx, req.token, req.ThingIDs...); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func identifyEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(identifyReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		id, err := svc.Identify(ctx, req.ThingKey)
		if err != nil {
			return nil, err
		}

		res := identityRes{
			ID: id,
		}

		return res, nil
	}
}

func updateExternalKeyEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(updateExternalKeyReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.UpdateExternalKey(ctx, req.token, req.Key, req.thingID); err != nil {
			return nil, err
		}

		return updateExternalKeyRes{}, nil
	}
}

func removeExternalKeyEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(resourceReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RemoveExternalKey(ctx, req.token, req.id); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func buildThingsResponse(tp things.ThingsPage, pm apiutil.PageMetadata) ThingsPageRes {
	res := ThingsPageRes{
		pageRes: pageRes{
			Total:  tp.Total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
			Order:  pm.Order,
			Dir:    pm.Dir,
			Name:   pm.Name,
		},
		Things: []thingRes{},
	}

	for _, t := range tp.Things {
		view := thingRes{
			ID:        t.ID,
			GroupID:   t.GroupID,
			ProfileID: t.ProfileID,
			Name:      t.Name,
			Key:       t.Key,
			Metadata:  t.Metadata,
		}
		res.Things = append(res.Things, view)
	}

	return res
}
