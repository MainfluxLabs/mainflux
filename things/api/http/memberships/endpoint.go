// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package memberships

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/go-kit/kit/endpoint"
)

func createGroupMembershipsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(groupMembershipsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		var gms []things.GroupMembership
		for _, g := range req.GroupMemberships {
			gp := things.GroupMembership{
				MemberID: g.MemberID,
				GroupID:  req.groupID,
				Role:     g.Role,
			}
			gms = append(gms, gp)
		}

		if err := svc.CreateGroupMemberships(ctx, req.token, gms...); err != nil {
			return nil, err
		}

		return createGroupMembershipsRes{}, nil
	}
}

func listGroupMembershipsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listGroupMembershipsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		gmp, err := svc.ListGroupMemberships(ctx, req.token, req.groupID, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildGroupMembershipsResponse(gmp, req.pageMetadata), nil
	}
}

func updateGroupMembershipsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(groupMembershipsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		var gms []things.GroupMembership
		for _, g := range req.GroupMemberships {
			gp := things.GroupMembership{
				MemberID: g.MemberID,
				GroupID:  req.groupID,
				Role:     g.Role,
			}
			gms = append(gms, gp)
		}

		if err := svc.UpdateGroupMemberships(ctx, req.token, gms...); err != nil {
			return nil, err
		}

		return updateGroupMembershipsRes{}, nil
	}
}

func removeGroupMembershipsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(removeGroupMembershipsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RemoveGroupMemberships(ctx, req.token, req.groupID, req.MemberIDs...); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func buildGroupMembershipsResponse(gpp things.GroupMembershipsPage, pm apiutil.PageMetadata) listGroupMembershipsRes {
	res := listGroupMembershipsRes{
		pageRes: pageRes{
			Total:  gpp.Total,
			Limit:  pm.Limit,
			Offset: pm.Offset,
			Email:  pm.Email,
			Order:  pm.Order,
			Dir:    pm.Dir,
		},
		GroupMemberships: []groupMembership{},
	}

	for _, g := range gpp.GroupMemberships {
		gp := groupMembership{
			Email:    g.Email,
			MemberID: g.MemberID,
			Role:     g.Role,
		}
		res.GroupMemberships = append(res.GroupMemberships, gp)
	}

	return res
}
