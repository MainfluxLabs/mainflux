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
			ValuesFilter: p.Transformer.ValuesFilter,
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

		return connByKeyRes{channelID: conn.ChannelID, thingID: conn.ThingID, profile: profile}, nil
	}
}

func authorizeEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(authorizeReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		ar := things.AuthorizeReq{
			Token:   req.token,
			Object:  req.object,
			Subject: req.subject,
			Action:  req.action,
		}

		if err := svc.Authorize(ctx, ar); err != nil {
			return emptyRes{}, err
		}

		return emptyRes{}, nil
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
				Name:        g.Name,
				Description: g.Description,
			}
			mgr = append(mgr, &gr)
		}

		return getGroupsByIDsRes{groups: mgr}, nil
	}
}

func getProfileByThingIDEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(profileByThingIDReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		p, err := svc.GetProfileByThingID(ctx, req.thingID)
		if err != nil {
			return profileByThingIDRes{}, err
		}

		transformer := &protomfx.Transformer{
			ValuesFilter: p.Transformer.ValuesFilter,
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

		return profileByThingIDRes{profile: profile}, nil
	}
}

func GetGroupIDByThingIDEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(groupIDByThingIDReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		groupID, err := svc.GetGroupIDByThingID(ctx, req.thingID)
		if err != nil {
			return groupIDByThingIDRes{}, err
		}

		return groupIDByThingIDRes{groupID: groupID}, nil
	}
}
