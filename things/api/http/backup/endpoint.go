// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package backup

import (
	"context"

	"github.com/MainfluxLabs/mainflux/things"
	"github.com/MainfluxLabs/mainflux/things/api/http/memberships"
	"github.com/go-kit/kit/endpoint"
)

func backupEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(backupReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		backup, err := svc.Backup(ctx, req.token)
		if err != nil {
			return nil, err
		}

		return buildBackupResponse(backup), nil
	}
}

func restoreEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(restoreReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		backup := buildBackup(req)

		if err := svc.Restore(ctx, req.token, backup); err != nil {
			return nil, err
		}

		return restoreRes{}, nil
	}
}

func buildBackupResponse(backup things.Backup) backupRes {
	res := backupRes{
		Things:           []viewThingRes{},
		Profiles:         []backupProfile{},
		Groups:           []backupGroup{},
		GroupMemberships: []memberships.ViewGroupMembershipRes{},
	}

	for _, thing := range backup.Things {
		view := viewThingRes{
			ID:          thing.ID,
			GroupID:     thing.GroupID,
			ProfileID:   thing.ProfileID,
			Name:        thing.Name,
			Key:         thing.Key,
			ExternalKey: thing.ExternalKey,
			Metadata:    thing.Metadata,
		}

		res.Things = append(res.Things, view)
	}

	for _, profile := range backup.Profiles {
		view := backupProfile{
			ID:       profile.ID,
			GroupID:  profile.GroupID,
			Name:     profile.Name,
			Config:   profile.Config,
			Metadata: profile.Metadata,
		}
		res.Profiles = append(res.Profiles, view)
	}

	for _, group := range backup.Groups {
		view := backupGroup{
			ID:          group.ID,
			Name:        group.Name,
			OrgID:       group.OrgID,
			Description: group.Description,
			Metadata:    group.Metadata,
			CreatedAt:   group.CreatedAt,
			UpdatedAt:   group.UpdatedAt,
		}
		res.Groups = append(res.Groups, view)
	}

	for _, membership := range backup.GroupMemberships {
		view := memberships.ViewGroupMembershipRes{
			MemberID: membership.MemberID,
			GroupID:  membership.GroupID,
			Email:    membership.Email,
			Role:     membership.Role,
		}
		res.GroupMemberships = append(res.GroupMemberships, view)
	}

	return res
}

func buildBackup(req restoreReq) (backup things.Backup) {
	for _, thing := range req.Things {
		th := things.Thing{
			ID:          thing.ID,
			GroupID:     thing.GroupID,
			ProfileID:   thing.ProfileID,
			Name:        thing.Name,
			Key:         thing.Key,
			ExternalKey: thing.ExternalKey,
			Metadata:    thing.Metadata,
		}
		backup.Things = append(backup.Things, th)
	}

	for _, profile := range req.Profiles {
		pr := things.Profile{
			ID:       profile.ID,
			GroupID:  profile.GroupID,
			Name:     profile.Name,
			Config:   profile.Config,
			Metadata: profile.Metadata,
		}
		backup.Profiles = append(backup.Profiles, pr)
	}

	for _, group := range req.Groups {
		gr := things.Group{
			ID:          group.ID,
			Name:        group.Name,
			OrgID:       group.OrgID,
			Description: group.Description,
			Metadata:    group.Metadata,
			CreatedAt:   group.CreatedAt,
			UpdatedAt:   group.UpdatedAt,
		}
		backup.Groups = append(backup.Groups, gr)
	}

	for _, membership := range req.GroupMemberships {
		gm := things.GroupMembership{
			GroupID:  membership.GroupID,
			MemberID: membership.MemberID,
			Email:    membership.Email,
			Role:     membership.Role,
		}
		backup.GroupMemberships = append(backup.GroupMemberships, gm)
	}

	return backup
}
