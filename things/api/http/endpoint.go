// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"

	"github.com/MainfluxLabs/mainflux/things"
	"github.com/go-kit/kit/endpoint"
)

func createThingsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createThingsReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		ths := []things.Thing{}
		for _, t := range req.Things {
			th := things.Thing{
				ID:        t.ID,
				GroupID:   req.groupID,
				ProfileID: t.ProfileID,
				Name:      t.Name,
				Key:       t.Key,
				Metadata:  t.Metadata,
			}
			ths = append(ths, th)
		}

		saved, err := svc.CreateThings(ctx, req.token, ths...)
		if err != nil {
			return nil, err
		}

		res := thingsRes{
			Things:  []thingRes{},
			created: true,
		}

		for _, t := range saved {
			th := thingRes{
				ID:        t.ID,
				GroupID:   t.GroupID,
				ProfileID: t.ProfileID,
				Name:      t.Name,
				Key:       t.Key,
				Metadata:  t.Metadata,
			}
			res.Things = append(res.Things, th)
		}

		return res, nil
	}
}

func updateThingEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateThingReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		thing := things.Thing{
			ID:        req.id,
			ProfileID: req.ProfileID,
			Name:      req.Name,
			Metadata:  req.Metadata,
		}

		if err := svc.UpdateThing(ctx, req.token, thing); err != nil {
			return nil, err
		}

		res := thingRes{ID: req.id, created: false}
		return res, nil
	}
}

func updateThingsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateThingsReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		ths := []things.Thing{}
		for _, t := range req.Things {
			th := things.Thing{
				ID:        t.ID,
				ProfileID: t.ProfileID,
				Name:      t.Name,
				Metadata:  t.Metadata,
			}
			ths = append(ths, th)
		}

		if err := svc.UpdateThings(ctx, req.token, ths...); err != nil {
			return nil, err
		}

		res := thingsRes{
			created: false,
		}
		return res, nil
	}
}

func updateKeyEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateKeyReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.UpdateKey(ctx, req.token, req.id, req.Key); err != nil {
			return nil, err
		}

		res := thingRes{ID: req.id, created: false}
		return res, nil
	}
}

func viewThingEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(resourceReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		thing, err := svc.ViewThing(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		res := viewThingRes{
			ID:        thing.ID,
			GroupID:   thing.GroupID,
			ProfileID: thing.ProfileID,
			Name:      thing.Name,
			Key:       thing.Key,
			Metadata:  thing.Metadata,
		}
		return res, nil
	}
}

func viewMetadataByKeyEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewMetadataReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		metadata, err := svc.ViewMetadataByKey(ctx, req.key)
		if err != nil {
			return nil, err
		}

		res := viewMetadataRes{
			Metadata: metadata,
		}

		return res, nil
	}
}

func listThingsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listResourcesReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListThings(ctx, req.token, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildThingsResponse(page), nil
	}
}

func listThingsByProfileEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listByIDReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListThingsByProfile(ctx, req.token, req.id, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildThingsResponse(page), nil
	}
}

func removeThingEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(resourceReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RemoveThings(ctx, req.token, req.id); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func removeThingsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(removeThingsReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RemoveThings(ctx, req.token, req.ThingIDs...); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func createProfilesEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createProfilesReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		prs := []things.Profile{}
		for _, c := range req.Profiles {
			pr := things.Profile{
				Name:     c.Name,
				ID:       c.ID,
				Config:   c.Config,
				GroupID:  req.groupID,
				Metadata: c.Metadata,
			}
			prs = append(prs, pr)
		}

		saved, err := svc.CreateProfiles(ctx, req.token, prs...)
		if err != nil {
			return nil, err
		}

		res := profilesRes{
			Profiles: []profileRes{},
			created:  true,
		}

		for _, c := range saved {
			pr := profileRes{
				ID:       c.ID,
				Name:     c.Name,
				GroupID:  c.GroupID,
				Config:   c.Config,
				Metadata: c.Metadata,
			}
			res.Profiles = append(res.Profiles, pr)
		}

		return res, nil
	}
}

func updateProfileEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateProfileReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		profile := things.Profile{
			ID:       req.id,
			Name:     req.Name,
			Config:   req.Config,
			Metadata: req.Metadata,
		}
		if err := svc.UpdateProfile(ctx, req.token, profile); err != nil {
			return nil, err
		}

		res := profileRes{
			ID:      req.id,
			created: false,
		}
		return res, nil
	}
}

func viewProfileEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(resourceReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		pr, err := svc.ViewProfile(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		res := profileRes{
			ID:       pr.ID,
			GroupID:  pr.GroupID,
			Name:     pr.Name,
			Metadata: pr.Metadata,
			Config:   pr.Config,
		}

		return res, nil
	}
}

func listProfilesEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listResourcesReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListProfiles(ctx, req.token, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildProfilesResponse(page), nil
	}
}

func viewProfileByThingEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(resourceReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		pr, err := svc.ViewProfileByThing(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		res := profileRes{
			ID:       pr.ID,
			GroupID:  pr.GroupID,
			Name:     pr.Name,
			Config:   pr.Config,
			Metadata: pr.Metadata,
		}

		return res, nil
	}
}

func removeProfileEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(resourceReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RemoveProfiles(ctx, req.token, req.id); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func removeProfilesEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(removeProfilesReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RemoveProfiles(ctx, req.token, req.ProfileIDs...); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func backupEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
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
	return func(ctx context.Context, request interface{}) (interface{}, error) {
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

func createGroupsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createGroupsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		grs := []things.Group{}
		for _, g := range req.Groups {
			group := things.Group{
				Name:        g.Name,
				OrgID:       req.orgID,
				Description: g.Description,
				Metadata:    g.Metadata,
			}
			grs = append(grs, group)
		}

		groups, err := svc.CreateGroups(ctx, req.token, grs...)
		if err != nil {
			return nil, err
		}

		res := groupsRes{
			Groups:  []groupRes{},
			created: true,
		}

		for _, gr := range groups {
			gRes := groupRes{
				ID:          gr.ID,
				OrgID:       gr.OrgID,
				Name:        gr.Name,
				Description: gr.Description,
				Metadata:    gr.Metadata,
			}
			res.Groups = append(res.Groups, gRes)
		}

		return res, nil
	}
}

func viewGroupEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(resourceReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		group, err := svc.ViewGroup(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		res := viewGroupRes{
			ID:          group.ID,
			Name:        group.Name,
			Description: group.Description,
			Metadata:    group.Metadata,
			OrgID:       group.OrgID,
			CreatedAt:   group.CreatedAt,
			UpdatedAt:   group.UpdatedAt,
		}

		return res, nil
	}
}

func updateGroupEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateGroupReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		group := things.Group{
			ID:          req.id,
			Name:        req.Name,
			Description: req.Description,
			Metadata:    req.Metadata,
		}

		_, err := svc.UpdateGroup(ctx, req.token, group)
		if err != nil {
			return nil, err
		}

		res := groupRes{created: false}
		return res, nil
	}
}

func removeGroupEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(resourceReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RemoveGroups(ctx, req.token, req.id); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func removeGroupsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(removeGroupsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RemoveGroups(ctx, req.token, req.GroupIDs...); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func listGroupsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listGroupsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListGroups(ctx, req.token, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildGroupsResponse(page), nil
	}
}

func listGroupsByOrgEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listByIDReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListGroupsByOrg(ctx, req.token, req.id, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildGroupsResponse(page), nil
	}
}

func listThingsByOrgEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listByIDReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListThingsByOrg(ctx, req.token, req.id, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildThingsResponse(page), nil
	}
}

func listThingsByGroupEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listByIDReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListThingsByGroup(ctx, req.token, req.id, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildThingsResponse(page), nil
	}
}

func viewGroupByThingEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(resourceReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		group, err := svc.ViewGroupByThing(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		groupRes := viewGroupRes{
			ID:          group.ID,
			Name:        group.Name,
			Description: group.Description,
			Metadata:    group.Metadata,
			OrgID:       group.OrgID,
			CreatedAt:   group.CreatedAt,
			UpdatedAt:   group.UpdatedAt,
		}

		return groupRes, nil
	}
}

func listProfilesByOrgEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listByIDReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListProfilesByOrg(ctx, req.token, req.id, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildProfilesResponse(page), nil
	}
}

func listProfilesByGroupEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listByIDReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListProfilesByGroup(ctx, req.token, req.id, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildProfilesResponse(page), nil
	}
}

func viewGroupByProfileEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(resourceReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		group, err := svc.ViewGroupByProfile(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		groupRes := viewGroupRes{
			ID:          group.ID,
			Name:        group.Name,
			Description: group.Description,
			Metadata:    group.Metadata,
			OrgID:       group.OrgID,
			CreatedAt:   group.CreatedAt,
			UpdatedAt:   group.UpdatedAt,
		}

		return groupRes, nil
	}
}

func buildGroupsResponse(gp things.GroupPage) groupPageRes {
	res := groupPageRes{
		pageRes: pageRes{
			Total:  gp.Total,
			Limit:  gp.Limit,
			Offset: gp.Offset,
		},
		Groups: []viewGroupRes{},
	}

	for _, group := range gp.Groups {
		view := viewGroupRes{
			ID:          group.ID,
			OrgID:       group.OrgID,
			Name:        group.Name,
			Description: group.Description,
			Metadata:    group.Metadata,
			CreatedAt:   group.CreatedAt,
			UpdatedAt:   group.UpdatedAt,
		}
		res.Groups = append(res.Groups, view)
	}

	return res
}

func buildThingsResponse(tp things.ThingsPage) ThingsPageRes {
	res := ThingsPageRes{
		pageRes: pageRes{
			Total:  tp.Total,
			Offset: tp.Offset,
			Limit:  tp.Limit,
			Order:  tp.Order,
			Dir:    tp.Dir,
			Name:   tp.Name,
		},
		Things: []thingRes{},
	}

	for _, t := range tp.Things {
		view := thingRes{
			ID:        t.ID,
			GroupID:   t.GroupID,
			ProfileID: t.ProfileID,
			Name:      t.Name,
			Key:       t.Key,
			Metadata:  t.Metadata,
		}
		res.Things = append(res.Things, view)
	}

	return res
}

func buildProfilesResponse(pp things.ProfilesPage) profilesPageRes {
	res := profilesPageRes{
		pageRes: pageRes{
			Total:  pp.Total,
			Offset: pp.Offset,
			Limit:  pp.Limit,
			Order:  pp.Order,
			Dir:    pp.Dir,
			Name:   pp.Name,
		},
		Profiles: []profileRes{},
	}

	for _, pr := range pp.Profiles {
		c := profileRes{
			ID:       pr.ID,
			GroupID:  pr.GroupID,
			Name:     pr.Name,
			Config:   pr.Config,
			Metadata: pr.Metadata,
		}
		res.Profiles = append(res.Profiles, c)
	}

	return res
}

func buildBackupResponse(backup things.Backup) backupRes {
	res := backupRes{
		Things:   []backupThingRes{},
		Profiles: []backupProfileRes{},
		Groups:   []viewGroupRes{},
	}

	for _, thing := range backup.Things {
		view := backupThingRes{
			ID:        thing.ID,
			GroupID:   thing.GroupID,
			ProfileID: thing.ProfileID,
			Name:      thing.Name,
			Key:       thing.Key,
			Metadata:  thing.Metadata,
		}
		res.Things = append(res.Things, view)
	}

	for _, profile := range backup.Profiles {
		view := backupProfileRes{
			ID:       profile.ID,
			GroupID:  profile.GroupID,
			Name:     profile.Name,
			Config:   profile.Config,
			Metadata: profile.Metadata,
		}
		res.Profiles = append(res.Profiles, view)
	}

	for _, group := range backup.Groups {
		view := viewGroupRes{
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

	return res
}

func buildBackup(req restoreReq) (backup things.Backup) {
	for _, thing := range req.Things {
		th := things.Thing{
			ID:       thing.ID,
			Name:     thing.Name,
			Key:      thing.Key,
			Metadata: thing.Metadata,
		}
		backup.Things = append(backup.Things, th)
	}

	for _, profile := range req.Profiles {
		pr := things.Profile{
			ID:       profile.ID,
			Name:     profile.Name,
			Metadata: profile.Metadata,
		}
		backup.Profiles = append(backup.Profiles, pr)
	}

	for _, group := range req.Groups {
		gr := things.Group{
			ID:          group.ID,
			Name:        group.Name,
			Description: group.Description,
			Metadata:    group.Metadata,
			CreatedAt:   group.CreatedAt,
			UpdatedAt:   group.UpdatedAt,
		}
		backup.Groups = append(backup.Groups, gr)
	}

	return backup
}

func identifyEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(identifyReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		id, err := svc.Identify(ctx, req.Token)
		if err != nil {
			return nil, err
		}

		res := identityRes{
			ID: id,
		}

		return res, nil
	}
}

func createRolesByGroupEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(groupRolesReq)
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

		if err := svc.CreateRolesByGroup(ctx, req.token, gms...); err != nil {
			return nil, err
		}

		return createGroupRolesRes{}, nil
	}
}

func updateRolesByGroupEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(groupRolesReq)
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

		if err := svc.UpdateRolesByGroup(ctx, req.token, gms...); err != nil {
			return nil, err
		}

		return updateGroupRolesRes{}, nil
	}
}

func removeRolesByGroupEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(removeGroupRolesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RemoveRolesByGroup(ctx, req.token, req.groupID, req.MemberIDs...); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func listRolesByGroupEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listByIDReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		gpp, err := svc.ListRolesByGroup(ctx, req.token, req.id, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildGroupRolesResponse(gpp), nil
	}
}

func buildGroupRolesResponse(gpp things.GroupMembersPage) listGroupRolesRes {
	res := listGroupRolesRes{
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
