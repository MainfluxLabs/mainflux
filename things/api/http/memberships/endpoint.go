// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package memberships

import (
	"context"
	"encoding/json"
	"fmt"

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

func backupGroupMembershipsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(backupByGroupReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		backup, err := svc.BackupGroupMemberships(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		fileName := fmt.Sprintf("group-memberships-backup-%s.json", req.id)
		return buildBackupResponse(backup, fileName)
	}
}

func restoreGroupMembershipsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(restoreByGroupReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		groupMembershipsBackup := buildGroupMembershipsBackup(req.GroupMemberships)

		if err := svc.RestoreGroupMemberships(ctx, req.token, req.id, groupMembershipsBackup); err != nil {
			return nil, err
		}

		return restoreRes{}, nil
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

func buildGroupMembershipsBackup(groupMemberships []ViewGroupMembershipRes) (backup things.GroupMembershipsBackup) {
	for _, membership := range groupMemberships {
		gm := things.GroupMembership{
			MemberID: membership.MemberID,
			GroupID:  membership.GroupID,
			Email:    membership.Email,
			Role:     membership.Role,
		}
		backup.GroupMemberships = append(backup.GroupMemberships, gm)
	}
	return backup
}

func buildBackupResponse(b things.GroupMembershipsBackup, fileName string) (apiutil.ViewFileRes, error) {
	views := make([]ViewGroupMembershipRes, 0, len(b.GroupMemberships))
	for _, membership := range b.GroupMemberships {
		views = append(views, ViewGroupMembershipRes{
			MemberID: membership.MemberID,
			GroupID:  membership.GroupID,
			Email:    membership.Email,
			Role:     membership.Role,
		})
	}

	data, err := json.MarshalIndent(views, "", "  ")
	if err != nil {
		return apiutil.ViewFileRes{}, err
	}

	return apiutil.ViewFileRes{
		File:     data,
		FileName: fileName,
	}, nil
}
