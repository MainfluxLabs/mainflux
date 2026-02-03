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

func getPubConfigByKeyEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(thingKey)
		if err := req.validate(); err != nil {
			return nil, err
		}

		pc, err := svc.GetPubConfigByKey(ctx, things.ThingKey{Type: req.keyType, Value: req.value})
		if err != nil {
			return pubConfigByKeyRes{}, err
		}

		config, err := buildConfigResponse(pc.ProfileConfig)
		if err != nil {
			return pubConfigByKeyRes{}, err
		}

		res := pubConfigByKeyRes{
			publisherID:   pc.PublisherID,
			profileConfig: config,
		}

		return res, nil
	}
}

func getConfigByThingEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(thingIDReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		c, err := svc.GetConfigByThing(ctx, req.thingID)
		if err != nil {
			return configByThingRes{}, err
		}

		config, err := buildConfigResponse(c)
		if err != nil {
			return pubConfigByKeyRes{}, err
		}

		return configByThingRes{config: config}, nil
	}
}

func canUserAccessThingEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
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
	return func(ctx context.Context, request any) (any, error) {
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
	return func(ctx context.Context, request any) (any, error) {
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
	return func(ctx context.Context, request any) (any, error) {
		req := request.(thingAccessGroupReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		r := things.ThingAccessReq{
			ThingKey: things.ThingKey{
				Value: req.thingKey.value,
				Type:  req.keyType,
			},
			ID: req.id,
		}

		if err := svc.CanThingAccessGroup(ctx, r); err != nil {
			return emptyRes{}, err
		}

		return emptyRes{}, nil
	}
}

func identifyEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(thingKey)
		if err := req.validate(); err != nil {
			return nil, err
		}

		id, err := svc.Identify(ctx, things.ThingKey{Type: req.keyType, Value: req.value})
		if err != nil {
			return identityRes{}, err
		}

		return identityRes{id: id}, nil
	}
}

func getGroupIDByThingEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(thingIDReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		groupID, err := svc.GetGroupIDByThing(ctx, req.thingID)
		if err != nil {
			return groupIDRes{}, err
		}

		return groupIDRes{groupID: groupID}, nil
	}
}

func getGroupIDByProfileEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(profileIDReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		groupID, err := svc.GetGroupIDByProfile(ctx, req.profileID)
		if err != nil {
			return groupIDRes{}, err
		}

		return groupIDRes{groupID: groupID}, nil
	}
}

func buildConfigResponse(conf map[string]any) (*protomfx.Config, error) {
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
		Transformer: transformer,
	}

	return profileConfig, nil
}

func getGroupIDsByOrgEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(orgAccessReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		groupIDs, err := svc.GetGroupIDsByOrg(ctx, req.orgID, req.token)
		if err != nil {
			return groupIDsRes{}, err
		}

		return groupIDsRes{groupIDs: groupIDs}, nil
	}
}

func getThingIDsByProfileEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(profileIDReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		thingIDs, err := svc.GetThingIDsByProfile(ctx, req.profileID)
		if err != nil {
			return thingIDsRes{}, err
		}

		return thingIDsRes{thingIDs: thingIDs}, nil
	}
}

func createGroupMembershipsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(createGroupMembershipsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		gms := make([]things.GroupMembership, 0, len(req.memberships))
		for _, memb := range req.memberships {
			gms = append(gms, things.GroupMembership{
				GroupID:  memb.groupID,
				MemberID: memb.userID,
				Role:     memb.role,
			})
		}

		if err := svc.CreateGroupMembershipsInternal(ctx, gms...); err != nil {
			return emptyRes{}, err
		}

		return emptyRes{}, nil
	}
}

func getGroupEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(getGroupReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		group, err := svc.ViewGroupInternal(ctx, req.groupID)
		if err != nil {
			return groupRes{}, err
		}

		return groupRes{
			id:    group.ID,
			orgID: group.OrgID,
			name:  group.Name,
		}, nil
	}
}
