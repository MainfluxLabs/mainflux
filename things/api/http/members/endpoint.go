// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package members

import (
	"context"

	"github.com/MainfluxLabs/mainflux/things"
	"github.com/go-kit/kit/endpoint"
)

func createGroupMembersEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(groupMembersReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		var gms []things.GroupMember
		for _, g := range req.GroupMembers {
			gp := things.GroupMember{
				MemberID: g.ID,
				GroupID:  req.groupID,
				Role:     g.Role,
			}
			gms = append(gms, gp)
		}

		if err := svc.CreateGroupMembers(ctx, req.token, gms...); err != nil {
			return nil, err
		}

		return createGroupMembersRes{}, nil
	}
}

func listGroupMembersEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listByGroupReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		gpp, err := svc.ListGroupMembers(ctx, req.token, req.id, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildGroupMembersResponse(gpp), nil
	}
}

func updateGroupMembersEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(groupMembersReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		var gms []things.GroupMember
		for _, g := range req.GroupMembers {
			gp := things.GroupMember{
				MemberID: g.ID,
				GroupID:  req.groupID,
				Role:     g.Role,
			}
			gms = append(gms, gp)
		}

		if err := svc.UpdateGroupMembers(ctx, req.token, gms...); err != nil {
			return nil, err
		}

		return updateGroupMembersRes{}, nil
	}
}

func removeGroupMembersEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(removeGroupMembersReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RemoveGroupMembers(ctx, req.token, req.groupID, req.MemberIDs...); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func buildGroupMembersResponse(gpp things.GroupMembersPage) listGroupMembersRes {
	res := listGroupMembersRes{
		pageRes: pageRes{
			Total:  gpp.Total,
			Limit:  gpp.Limit,
			Offset: gpp.Offset,
		},
		GroupMembers: []groupMember{},
	}

	for _, g := range gpp.GroupMembers {
		gp := groupMember{
			Email: g.Email,
			ID:    g.MemberID,
			Role:  g.Role,
		}
		res.GroupMembers = append(res.GroupMembers, gp)
	}

	return res
}
