// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"encoding/json"

	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/go-kit/kit/endpoint"
)

func getPubConfByKeyEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(pubConfByKeyReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		pc, err := svc.GetPubConfByKey(ctx, req.key)
		if err != nil {
			return pubConfByKeyRes{}, err
		}

		return buildPubConfResponse(pc)
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
				OrgID:       g.OrgID,
				Name:        g.Name,
				Description: g.Description,
			}
			mgr = append(mgr, &gr)
		}

		return getGroupsByIDsRes{groups: mgr}, nil
	}
}

func getGroupIDByThingIDEndpoint(svc things.Service) endpoint.Endpoint {
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

func buildPubConfResponse(pc things.PubConfInfo) (pubConfByKeyRes, error) {
	cb, err := json.Marshal(pc.ProfileConfig)
	if err != nil {
		return pubConfByKeyRes{}, err
	}

	var config things.Config
	if err := json.Unmarshal(cb, &config); err != nil {
		return pubConfByKeyRes{}, err
	}

	transformer := &protomfx.Transformer{
		ValuesFilter: config.Transformer.ValuesFilter,
		TimeField:    config.Transformer.TimeField,
		TimeFormat:   config.Transformer.TimeFormat,
		TimeLocation: config.Transformer.TimeLocation,
	}

	profileConfig := &protomfx.Config{
		ContentType: config.ContentType,
		Write:       config.Write,
		Transformer: transformer,
		WebhookID:   config.WebhookID,
		SmtpID:      config.SmtpID,
		SmppID:      config.SmppID,
	}

	res := pubConfByKeyRes{
		profileID:     pc.ProfileID,
		thingID:       pc.ThingID,
		profileConfig: profileConfig,
	}

	return res, nil
}
