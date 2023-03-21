// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/go-kit/kit/endpoint"
)

func canAccessEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(accessByKeyReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		id, err := svc.CanAccessByKey(ctx, req.chanID, req.thingKey)
		if err != nil {
			return identityRes{}, err
		}
		return identityRes{id: id}, nil
	}
}

func canAccessByIDEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(accessByIDReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		err := svc.CanAccessByID(ctx, req.chanID, req.thingID)
		return emptyRes{err: err}, err
	}
}

func isChannelOwnerEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(channelOwnerReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		err := svc.IsChannelOwner(ctx, req.owner, req.chanID)
		return emptyRes{err: err}, err
	}
}

func identifyEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(identifyReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		id, err := svc.Identify(ctx, req.key)
		if err != nil {
			return identityRes{}, err
		}

		return identityRes{id: id}, nil
	}
}

func listGroupsByIDsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getGroupsByIDsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		groups, err := svc.ListGroupsByIDs(ctx, req.ids)
		if err != nil {
			return getGroupsByIDsRes{}, err
		}

		mgr := []*mainflux.Group{}

		for _, g := range groups {
			gr := mainflux.Group{
				Id:          g.ID,
				OwnerID:     g.OwnerID,
				Name:        g.Name,
				Description: g.Description,
			}
			mgr = append(mgr, &gr)
		}

		return getGroupsByIDsRes{groups: mgr}, nil
	}
}
