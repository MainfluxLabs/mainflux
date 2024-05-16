// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
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
				Name:     t.Name,
				Key:      t.Key,
				ID:       t.ID,
				GroupID:  req.groupID,
				Metadata: t.Metadata,
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
				ID:       t.ID,
				Name:     t.Name,
				Key:      t.Key,
				GroupID:  t.GroupID,
				Metadata: t.Metadata,
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
			ID:       req.id,
			Name:     req.Name,
			Metadata: req.Metadata,
		}

		if err := svc.UpdateThing(ctx, req.token, thing); err != nil {
			return nil, err
		}

		res := thingRes{ID: req.id, created: false}
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
			ID:       thing.ID,
			OwnerID:  thing.OwnerID,
			GroupID:  thing.GroupID,
			Name:     thing.Name,
			Key:      thing.Key,
			Metadata: thing.Metadata,
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

		res := thingsPageRes{
			pageRes: pageRes{
				Total:  page.Total,
				Offset: page.Offset,
				Limit:  page.Limit,
				Order:  page.Order,
				Dir:    page.Dir,
			},
			Things: []viewThingRes{},
		}
		for _, th := range page.Things {
			view := viewThingRes{
				ID:       th.ID,
				OwnerID:  th.OwnerID,
				GroupID:  th.GroupID,
				Name:     th.Name,
				Key:      th.Key,
				Metadata: th.Metadata,
			}
			res.Things = append(res.Things, view)
		}

		return res, nil
	}
}

func listThingsByChannelEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listByConnectionReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListThingsByChannel(ctx, req.token, req.id, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		res := thingsPageRes{
			pageRes: pageRes{
				Total:  page.Total,
				Offset: page.Offset,
				Limit:  page.Limit,
			},
			Things: []viewThingRes{},
		}
		for _, th := range page.Things {
			view := viewThingRes{
				ID:       th.ID,
				OwnerID:  th.OwnerID,
				GroupID:  th.GroupID,
				Key:      th.Key,
				Name:     th.Name,
				Metadata: th.Metadata,
			}
			res.Things = append(res.Things, view)
		}

		return res, nil
	}
}

func removeThingEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(resourceReq)

		if err := req.validate(); err != nil {
			if err == errors.ErrNotFound {
				return removeRes{}, nil
			}
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
			if err == errors.ErrNotFound {
				return removeRes{}, nil
			}
			return nil, err
		}

		if err := svc.RemoveThings(ctx, req.token, req.ThingIDs...); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func createChannelsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createChannelsReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		chs := []things.Channel{}
		for _, c := range req.Channels {
			ch := things.Channel{
				Name:     c.Name,
				ID:       c.ID,
				Profile:  c.Profile,
				GroupID:  req.groupID,
				Metadata: c.Metadata,
			}
			chs = append(chs, ch)
		}

		saved, err := svc.CreateChannels(ctx, req.token, chs...)
		if err != nil {
			return nil, err
		}

		res := channelsRes{
			Channels: []channelRes{},
			created:  true,
		}

		for _, c := range saved {
			ch := channelRes{
				ID:       c.ID,
				Name:     c.Name,
				GroupID:  c.GroupID,
				Profile:  c.Profile,
				Metadata: c.Metadata,
			}
			res.Channels = append(res.Channels, ch)
		}

		return res, nil
	}
}

func updateChannelEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateChannelReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		channel := things.Channel{
			ID:       req.id,
			Name:     req.Name,
			Profile:  req.Profile,
			Metadata: req.Metadata,
		}
		if err := svc.UpdateChannel(ctx, req.token, channel); err != nil {
			return nil, err
		}

		res := channelRes{
			ID:      req.id,
			created: false,
		}
		return res, nil
	}
}

func viewChannelEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(resourceReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		ch, err := svc.ViewChannel(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		res := channelRes{
			ID:       ch.ID,
			GroupID:  ch.GroupID,
			Name:     ch.Name,
			Metadata: ch.Metadata,
			Profile:  ch.Profile,
		}

		return res, nil
	}
}

func listChannelsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listResourcesReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListChannels(ctx, req.token, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		res := channelsPageRes{
			pageRes: pageRes{
				Total:  page.Total,
				Offset: page.Offset,
				Limit:  page.Limit,
				Order:  page.Order,
				Dir:    page.Dir,
			},
			Channels: []channelRes{},
		}
		// Cast channels
		for _, ch := range page.Channels {
			view := channelRes{
				ID:       ch.ID,
				GroupID:  ch.GroupID,
				Name:     ch.Name,
				Profile:  ch.Profile,
				Metadata: ch.Metadata,
			}

			res.Channels = append(res.Channels, view)
		}

		return res, nil
	}
}

func viewChannelByThingEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(resourceReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		ch, err := svc.ViewChannelByThing(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		res := channelRes{
			ID:       ch.ID,
			GroupID:  ch.GroupID,
			Name:     ch.Name,
			Profile:  ch.Profile,
			Metadata: ch.Metadata,
		}

		return res, nil
	}
}

func removeChannelEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(resourceReq)

		if err := req.validate(); err != nil {
			if err == errors.ErrNotFound {
				return removeRes{}, nil
			}
			return nil, err
		}

		if err := svc.RemoveChannels(ctx, req.token, req.id); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func removeChannelsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(removeChannelsReq)

		if err := req.validate(); err != nil {
			if err == errors.ErrNotFound {
				return removeRes{}, nil
			}
			return nil, err
		}

		if err := svc.RemoveChannels(ctx, req.token, req.ChannelIDs...); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func connectEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		cr := request.(connectionsReq)

		if err := cr.validate(); err != nil {
			return nil, err
		}

		if err := svc.Connect(ctx, cr.token, cr.ChannelID, cr.ThingIDs); err != nil {
			return nil, err
		}

		return connectionsRes{}, nil
	}
}

func disconnectEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		cr := request.(connectionsReq)
		if err := cr.validate(); err != nil {
			return nil, err
		}

		if err := svc.Disconnect(ctx, cr.token, cr.ChannelID, cr.ThingIDs); err != nil {
			return nil, err
		}

		return connectionsRes{}, nil
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
			OwnerID:     group.OwnerID,
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

		page, err := svc.ListGroups(ctx, req.token, req.orgID, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildGroupsResponse(page), nil
	}
}

func listThingsByGroupEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listMembersReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		pm := things.PageMetadata{
			Offset:   req.offset,
			Limit:    req.limit,
			Metadata: req.metadata,
		}

		page, err := svc.ListThingsByGroup(ctx, req.token, req.id, pm)
		if err != nil {
			return nil, err
		}

		return buildThingsByGroupResponse(page), nil
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
			OwnerID:     group.OwnerID,
			OrgID:       group.OrgID,
			CreatedAt:   group.CreatedAt,
			UpdatedAt:   group.UpdatedAt,
		}

		return groupRes, nil
	}
}

func listChannelsByGroupEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listMembersReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		pm := things.PageMetadata{
			Offset:   req.offset,
			Limit:    req.limit,
			Metadata: req.metadata,
		}

		page, err := svc.ListChannelsByGroup(ctx, req.token, req.id, pm)
		if err != nil {
			return nil, err
		}

		return buildChannelsByGroupResponse(page), nil
	}
}

func viewGroupByChannelEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(resourceReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		group, err := svc.ViewGroupByChannel(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		groupRes := viewGroupRes{
			ID:          group.ID,
			Name:        group.Name,
			Description: group.Description,
			Metadata:    group.Metadata,
			OwnerID:     group.OwnerID,
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
			OwnerID:     group.OwnerID,
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

func buildThingsByGroupResponse(tp things.ThingsPage) ThingsPageRes {
	res := ThingsPageRes{
		pageRes: pageRes{
			Total:  tp.Total,
			Offset: tp.Offset,
			Limit:  tp.Limit,
			Name:   tp.Name,
		},
		Things: []thingRes{},
	}

	for _, t := range tp.Things {
		view := thingRes{
			ID:       t.ID,
			Metadata: t.Metadata,
			Name:     t.Name,
			Key:      t.Key,
		}
		res.Things = append(res.Things, view)
	}

	return res
}

func buildChannelsByGroupResponse(cp things.ChannelsPage) channelsPageRes {
	res := channelsPageRes{
		pageRes: pageRes{
			Total:  cp.Total,
			Offset: cp.Offset,
			Limit:  cp.Limit,
			Name:   cp.Name,
		},
		Channels: []channelRes{},
	}

	for _, ch := range cp.Channels {
		c := channelRes{
			ID:       ch.ID,
			Profile:  ch.Profile,
			Metadata: ch.Metadata,
			Name:     ch.Name,
		}
		res.Channels = append(res.Channels, c)
	}

	return res
}

func buildBackupResponse(backup things.Backup) backupRes {
	res := backupRes{
		Things:                []backupThingRes{},
		Channels:              []backupChannelRes{},
		Connections:           []backupConnectionRes{},
		Groups:                []viewGroupRes{},
		GroupThingRelations:   []backupGroupThingRelationRes{},
		GroupChannelRelations: []backupGroupChannelRelationRes{},
	}

	for _, thing := range backup.Things {
		view := backupThingRes{
			ID:       thing.ID,
			Name:     thing.Name,
			OwnerID:  thing.OwnerID,
			Key:      thing.Key,
			Metadata: thing.Metadata,
		}
		res.Things = append(res.Things, view)
	}

	for _, channel := range backup.Channels {
		view := backupChannelRes{
			ID:       channel.ID,
			Name:     channel.Name,
			OwnerID:  channel.OwnerID,
			Metadata: channel.Metadata,
		}
		res.Channels = append(res.Channels, view)
	}

	for _, connection := range backup.Connections {
		view := backupConnectionRes{
			ChannelID: connection.ChannelID,
			ThingID:   connection.ThingID,
		}
		res.Connections = append(res.Connections, view)
	}

	for _, group := range backup.Groups {
		view := viewGroupRes{
			ID:          group.ID,
			Name:        group.Name,
			Description: group.Description,
			Metadata:    group.Metadata,
			OwnerID:     group.OwnerID,
			OrgID:       group.OrgID,
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
			OwnerID:  thing.OwnerID,
			Name:     thing.Name,
			Key:      thing.Key,
			Metadata: thing.Metadata,
		}
		backup.Things = append(backup.Things, th)
	}

	for _, channel := range req.Channels {
		ch := things.Channel{
			ID:       channel.ID,
			OwnerID:  channel.OwnerID,
			Name:     channel.Name,
			Metadata: channel.Metadata,
		}
		backup.Channels = append(backup.Channels, ch)
	}

	for _, connection := range req.Connections {
		conn := things.Connection{
			ChannelID: connection.ChannelID,
			ThingID:   connection.ThingID,
		}
		backup.Connections = append(backup.Connections, conn)
	}

	for _, group := range req.Groups {
		gr := things.Group{
			ID:          group.ID,
			OwnerID:     group.OwnerID,
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

func getConnByKeyEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getConnByKeyReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		conn, err := svc.GetConnByKey(ctx, req.Key)
		if err != nil {
			return nil, err
		}

		res := connByKeyRes{
			ChannelID: conn.ChannelID,
			ThingID:   conn.ThingID,
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

		var gps []things.GroupRoles
		for _, g := range req.GroupRoles {
			gp := things.GroupRoles{
				MemberID: g.ID,
				Role:     g.Role,
			}
			gps = append(gps, gp)
		}

		if err := svc.CreateRolesByGroup(ctx, req.token, req.groupID, gps...); err != nil {
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

		var gps []things.GroupRoles
		for _, g := range req.GroupRoles {
			gp := things.GroupRoles{
				MemberID: g.ID,
				Role:     g.Role,
			}
			gps = append(gps, gp)
		}

		if err := svc.UpdateRolesByGroup(ctx, req.token, req.groupID, gps...); err != nil {
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
		req := request.(listGroupMembersReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		pm := things.PageMetadata{
			Offset: req.offset,
			Limit:  req.limit,
		}

		gpp, err := svc.ListRolesByGroup(ctx, req.token, req.groupID, pm)
		if err != nil {
			return nil, err
		}

		return buildGroupRolesResponse(gpp), nil
	}
}

func buildGroupRolesResponse(gpp things.GroupRolesPage) listGroupRolesRes {
	res := listGroupRolesRes{
		pageRes: pageRes{
			Total:  gpp.Total,
			Limit:  gpp.Limit,
			Offset: gpp.Offset,
		},
		GroupRoles: []groupMember{},
	}

	for _, g := range gpp.GroupRoles {
		gp := groupMember{
			Email: g.Email,
			ID:    g.MemberID,
			Role:  g.Role,
		}
		res.GroupRoles = append(res.GroupRoles, gp)
	}

	return res
}
