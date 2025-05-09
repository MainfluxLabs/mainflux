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

		config, err := buildConfigResponse(pc.ProfileConfig)
		if err != nil {
			return pubConfByKeyRes{}, err
		}

		res := pubConfByKeyRes{
			publisherID:   pc.PublisherID,
			profileConfig: config,
		}

		return res, nil
	}
}

func getConfigByThingIDEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(thingIDReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		c, err := svc.GetConfigByThingID(ctx, req.thingID)
		if err != nil {
			return configByThingIDRes{}, err
		}

		config, err := buildConfigResponse(c)
		if err != nil {
			return pubConfByKeyRes{}, err
		}

		return configByThingIDRes{config: config}, nil
	}
}

func canUserAccessThingEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(userAccessThingReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		r := things.UserAccessReq{
			Token:  req.token,
			ID:     req.id,
			Action: req.action,
		}

		if err := svc.CanUserAccessThing(ctx, r); err != nil {
			return emptyRes{}, err
		}

		return emptyRes{}, nil
	}
}

func canUserAccessProfileEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(userAccessProfileReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		r := things.UserAccessReq{
			Token:  req.token,
			ID:     req.id,
			Action: req.action,
		}

		if err := svc.CanUserAccessProfile(ctx, r); err != nil {
			return emptyRes{}, err
		}

		return emptyRes{}, nil
	}
}

func canUserAccessGroupEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(userAccessGroupReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		r := things.UserAccessReq{
			Token:  req.token,
			ID:     req.id,
			Action: req.action,
		}

		if err := svc.CanUserAccessGroup(ctx, r); err != nil {
			return emptyRes{}, err
		}

		return emptyRes{}, nil
	}
}

func canThingAccessGroupEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(thingAccessGroupReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		r := things.ThingAccessReq{
			Key: req.key,
			ID:  req.id,
		}

		if err := svc.CanThingAccessGroup(ctx, r); err != nil {
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

func getGroupIDByThingIDEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(thingIDReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		groupID, err := svc.GetGroupIDByThingID(ctx, req.thingID)
		if err != nil {
			return groupIDRes{}, err
		}

		return groupIDRes{groupID: groupID}, nil
	}
}

func getGroupIDByProfileIDEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(profileIDReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		groupID, err := svc.GetGroupIDByProfileID(ctx, req.profileID)
		if err != nil {
			return groupIDRes{}, err
		}

		return groupIDRes{groupID: groupID}, nil
	}
}

func getProfileIDByThingIDEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(thingIDReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		profileID, err := svc.GetProfileIDByThingID(ctx, req.thingID)
		if err != nil {
			return profileIDRes{}, err
		}

		return profileIDRes{profileID: profileID}, nil
	}
}

func buildConfigResponse(conf map[string]interface{}) (*protomfx.Config, error) {
	cb, err := json.Marshal(conf)
	if err != nil {
		return &protomfx.Config{}, err
	}

	var config things.Config
	if err := json.Unmarshal(cb, &config); err != nil {
		return &protomfx.Config{}, err
	}

	transformer := &protomfx.Transformer{
		DataFilters:  config.Transformer.DataFilters,
		DataField:    config.Transformer.DataField,
		TimeField:    config.Transformer.TimeField,
		TimeFormat:   config.Transformer.TimeFormat,
		TimeLocation: config.Transformer.TimeLocation,
	}

	profileConfig := &protomfx.Config{
		ContentType: config.ContentType,
		Write:       config.Write,
		Transformer: transformer,
	}

	return profileConfig, nil
}
