// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/go-kit/kit/endpoint"
)

func getConnByKeyEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(connByKeyReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		conn, err := svc.GetConnByKey(ctx, req.key)
		if err != nil {
			return connByKeyRes{}, err
		}

		p, err := svc.ViewChannelProfile(ctx, conn.ChannelID)
		if err != nil {
			return connByKeyRes{}, err
		}

		transformer := &protomfx.Transformer{
			ValueFields:  p.Transformer.ValueFields,
			TimeField:    p.Transformer.TimeField,
			TimeFormat:   p.Transformer.TimeFormat,
			TimeLocation: p.Transformer.TimeLocation,
		}

		profile := &protomfx.Profile{
			ContentType: p.ContentType,
			Write:       p.Write,
			Transformer: transformer,
			WebhookID:   p.WebhookID,
			SmtpID:      p.SmtpID,
			SmppID:      p.SmppID,
		}

		return connByKeyRes{channelOD: conn.ChannelID, thingID: conn.ThingID, profile: profile}, nil
	}
}

func isChannelOwnerEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(channelOwnerReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		err := svc.IsChannelOwner(ctx, req.token, req.chanID)
		return emptyRes{err: err}, err
	}
}

func canAccessGroupEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(accessGroupReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		err := svc.CanAccessGroup(ctx, req.token, req.groupID, req.action)
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

		mgr := []*protomfx.Group{}

		for _, g := range groups {
			gr := protomfx.Group{
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
