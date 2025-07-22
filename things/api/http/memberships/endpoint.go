// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package memberships

import (
	"context"

	"github.com/MainfluxLabs/mainflux/things"
	"github.com/go-kit/kit/endpoint"
)

func createGroupMembershipsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
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
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listGroupMembershipsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		gmp, err := svc.ListGroupMemberships(ctx, req.token, req.groupID, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildGroupMembershipsResponse(gmp), nil
	}
}

func updateGroupMembershipsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
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
	return func(ctx context.Context, request interface{}) (interface{}, error) {
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

func backupGroupMembershipsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(backupReq)
		if err := req.validate(); err != nil {
			return nil, err
		}
		backup, err := svc.BackupGroupMemberships(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}
		return buildBackupResponse(backup), nil
	}
}

func buildGroupMembershipsResponse(gpp things.GroupMembershipsPage) listGroupMembershipsRes {
	res := listGroupMembershipsRes{
		pageRes: pageRes{
			Total:  gpp.Total,
			Limit:  gpp.Limit,
			Offset: gpp.Offset,
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

func buildBackupResponse(b things.BackupGroupMemberships) backupGroupMembershipsRes {
	res := backupGroupMembershipsRes{
		BackupGroupMemberships: []ViewGroupMembershipRes{},
	}
	for _, member := range b.BackupGroupMemberships {
		view := ViewGroupMembershipRes{
			MemberID: member.MemberID,
			GroupID:  member.GroupID,
			Email:    member.Email,
			Role:     member.Role,
		}
		res.BackupGroupMemberships = append(res.BackupGroupMemberships, view)
	}
	return res
}
