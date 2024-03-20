// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	"github.com/MainfluxLabs/mainflux"
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

		notifier := &mainflux.Notifier{
			Protocol:  p.Notifier.Protocol,
			Contacts:  p.Notifier.Contacts,
			Subtopics: p.Notifier.Subtopics,
		}

		writer := &mainflux.Writer{
			Subtopics:    p.Writer.Subtopics,
			TimeName:     p.Writer.TimeName,
			TimeFormat:   p.Writer.TimeFormat,
			TimeLocation: p.Writer.TimeLocation,
		}

		profile := &mainflux.Profile{
			ContentType: p.ContentType,
			Write:       p.Write,
			Writer:      writer,
			Notifier:    notifier,
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

		err := svc.IsChannelOwner(ctx, req.owner, req.chanID)
		return emptyRes{err: err}, err
	}
}

func isThingOwnerEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(thingOwnerReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		err := svc.IsThingOwner(ctx, req.token, req.thingID)
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
