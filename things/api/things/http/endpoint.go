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
		for _, tReq := range req.Things {
			th := things.Thing{
				Name:     tReq.Name,
				Key:      tReq.Key,
				ID:       tReq.ID,
				Metadata: tReq.Metadata,
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

		for _, th := range saved {
			tRes := thingRes{
				ID:       th.ID,
				Name:     th.Name,
				Key:      th.Key,
				Metadata: th.Metadata,
			}
			res.Things = append(res.Things, tRes)
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
		req := request.(viewResourceReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		thing, err := svc.ViewThing(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		res := viewThingRes{
			ID:       thing.ID,
			Owner:    thing.Owner,
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

		page, err := svc.ListThings(ctx, req.token, req.admin, req.pageMetadata)
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
		for _, thing := range page.Things {
			view := viewThingRes{
				ID:       thing.ID,
				Owner:    thing.Owner,
				Name:     thing.Name,
				Key:      thing.Key,
				Metadata: thing.Metadata,
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
		for _, thing := range page.Things {
			view := viewThingRes{
				ID:       thing.ID,
				Owner:    thing.Owner,
				Key:      thing.Key,
				Name:     thing.Name,
				Metadata: thing.Metadata,
			}
			res.Things = append(res.Things, view)
		}

		return res, nil
	}
}

func removeThingEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewResourceReq)

		err := req.validate()
		if err == errors.ErrNotFound {
			return removeRes{}, nil
		}

		if err != nil {
			return nil, err
		}

		if err := svc.RemoveThing(ctx, req.token, req.id); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func removeThingsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(removeThingsReq)

		err := req.validate()
		if err == errors.ErrNotFound {
			return removeRes{}, nil
		}

		if err != nil {
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
		for _, cReq := range req.Channels {
			ch := things.Channel{
				Metadata: cReq.Metadata,
				Name:     cReq.Name,
				ID:       cReq.ID,
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

		for _, ch := range saved {
			cRes := channelRes{
				ID:       ch.ID,
				Name:     ch.Name,
				Metadata: ch.Metadata,
			}
			res.Channels = append(res.Channels, cRes)
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
		req := request.(viewResourceReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		channel, err := svc.ViewChannel(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		res := viewChannelRes{
			ID:       channel.ID,
			Owner:    channel.Owner,
			Name:     channel.Name,
			Metadata: channel.Metadata,
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

		page, err := svc.ListChannels(ctx, req.token, req.admin, req.pageMetadata)
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
			Channels: []viewChannelRes{},
		}
		// Cast channels
		for _, channel := range page.Channels {
			view := viewChannelRes{
				ID:       channel.ID,
				Owner:    channel.Owner,
				Name:     channel.Name,
				Metadata: channel.Metadata,
			}

			res.Channels = append(res.Channels, view)
		}

		return res, nil
	}
}

func viewChannelByThingEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewResourceReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		ch, err := svc.ViewChannelByThing(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		res := viewChannelRes{
			ID:       ch.ID,
			Owner:    ch.Owner,
			Name:     ch.Name,
			Metadata: ch.Metadata,
		}

		return res, nil
	}
}

func removeChannelEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewResourceReq)

		if err := req.validate(); err != nil {
			if err == errors.ErrNotFound {
				return removeRes{}, nil
			}
			return nil, err
		}

		if err := svc.RemoveChannel(ctx, req.token, req.id); err != nil {
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
		for _, gReq := range req.Groups {
			group := things.Group{
				Name:        gReq.Name,
				Description: gReq.Description,
				Metadata:    gReq.Metadata,
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
		req := request.(groupReq)
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

func deleteGroupEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(groupReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RemoveGroup(ctx, req.token, req.id); err != nil {
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

		page, err := svc.ListGroups(ctx, req.token, req.admin, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildGroupsResponse(page), nil
	}
}
func listGroupThingsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listMembersReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		pm := things.PageMetadata{
			Offset:     req.offset,
			Limit:      req.limit,
			Metadata:   req.metadata,
			Unassigned: req.unassigned,
		}

		page, err := svc.ListGroupThings(ctx, req.token, req.id, pm)
		if err != nil {
			return nil, err
		}

		return buildGroupThingsResponse(page), nil
	}
}

func viewThingMembershipEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listMembersReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		group, err := svc.ViewThingMembership(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		groupRes := viewGroupRes{
			ID:          group.ID,
			Name:        group.Name,
			Description: group.Description,
			Metadata:    group.Metadata,
			OwnerID:     group.OwnerID,
			CreatedAt:   group.CreatedAt,
			UpdatedAt:   group.UpdatedAt,
		}

		return groupRes, nil
	}
}

func assignThingsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(groupThingsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.AssignThing(ctx, req.token, req.groupID, req.Things...); err != nil {
			return nil, err
		}

		return assignRes{}, nil
	}
}

func unassignThingsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(groupThingsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.UnassignThing(ctx, req.token, req.groupID, req.Things...); err != nil {
			return nil, err
		}

		return unassignRes{}, nil
	}
}

func listGroupChannelsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listMembersReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		pm := things.PageMetadata{
			Offset:     req.offset,
			Limit:      req.limit,
			Metadata:   req.metadata,
			Unassigned: req.unassigned,
		}

		page, err := svc.ListGroupChannels(ctx, req.token, req.id, pm)
		if err != nil {
			return nil, err
		}

		return buildGroupChannelsResponse(page), nil
	}
}

func listGroupThingsByChannelEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listGroupThingsByChannelReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListGroupThingsByChannel(ctx, req.token, req.groupID, req.channelID, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildGroupThingsResponse(page), nil
	}
}

func viewChannelMembershipEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listMembersReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		group, err := svc.ViewChannelMembership(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		groupRes := viewGroupRes{
			ID:          group.ID,
			Name:        group.Name,
			Description: group.Description,
			Metadata:    group.Metadata,
			OwnerID:     group.OwnerID,
			CreatedAt:   group.CreatedAt,
			UpdatedAt:   group.UpdatedAt,
		}

		return groupRes, nil
	}
}

func assignChannelsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(groupChannelsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.AssignChannel(ctx, req.token, req.groupID, req.Channels...); err != nil {
			return nil, err
		}

		return assignRes{}, nil
	}
}

func unassignChannelsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(groupChannelsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.UnassignChannel(ctx, req.token, req.groupID, req.Channels...); err != nil {
			return nil, err
		}

		return unassignRes{}, nil
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

func buildGroupThingsResponse(tp things.GroupThingsPage) groupThingsPageRes {
	res := groupThingsPageRes{
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

func buildGroupChannelsResponse(cp things.GroupChannelsPage) groupChannelsPageRes {
	res := groupChannelsPageRes{
		pageRes: pageRes{
			Total:  cp.Total,
			Offset: cp.Offset,
			Limit:  cp.Limit,
			Name:   cp.Name,
		},
		Channels: []channelRes{},
	}

	for _, c := range cp.Channels {
		view := channelRes{
			ID:       c.ID,
			Metadata: c.Metadata,
			Name:     c.Name,
		}
		res.Channels = append(res.Channels, view)
	}

	return res
}

func buildBackupResponse(backup things.Backup) backupRes {
	res := backupRes{
		Things:              []backupThingRes{},
		Channels:            []backupChannelRes{},
		Connections:         []backupConnectionRes{},
		Groups:              []viewGroupRes{},
		GroupThingRelations: []backupGroupThingRelationRes{},
	}

	for _, thing := range backup.Things {
		view := backupThingRes{
			ID:       thing.ID,
			Name:     thing.Name,
			Owner:    thing.Owner,
			Key:      thing.Key,
			Metadata: thing.Metadata,
		}
		res.Things = append(res.Things, view)
	}

	for _, channel := range backup.Channels {
		view := backupChannelRes{
			ID:       channel.ID,
			Name:     channel.Name,
			Owner:    channel.Owner,
			Metadata: channel.Metadata,
		}
		res.Channels = append(res.Channels, view)
	}

	for _, connection := range backup.Connections {
		view := backupConnectionRes{
			ChannelID:    connection.ChannelID,
			ChannelOwner: connection.ChannelOwner,
			ThingID:      connection.ThingID,
			ThingOwner:   connection.ThingOwner,
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
			CreatedAt:   group.CreatedAt,
			UpdatedAt:   group.UpdatedAt,
		}
		res.Groups = append(res.Groups, view)
	}

	for _, gtr := range backup.GroupThingRelations {
		view := backupGroupThingRelationRes{
			ThingID:   gtr.ThingID,
			GroupID:   gtr.GroupID,
			CreatedAt: gtr.CreatedAt,
			UpdatedAt: gtr.UpdatedAt,
		}
		res.GroupThingRelations = append(res.GroupThingRelations, view)
	}

	return res
}

func buildBackup(req restoreReq) (backup things.Backup) {
	for _, thing := range req.Things {
		th := things.Thing{
			ID:       thing.ID,
			Owner:    thing.Owner,
			Name:     thing.Name,
			Key:      thing.Key,
			Metadata: thing.Metadata,
		}
		backup.Things = append(backup.Things, th)
	}

	for _, channel := range req.Channels {
		ch := things.Channel{
			ID:       channel.ID,
			Owner:    channel.Owner,
			Name:     channel.Name,
			Metadata: channel.Metadata,
		}
		backup.Channels = append(backup.Channels, ch)
	}

	for _, connection := range req.Connections {
		conn := things.Connection{
			ChannelID:    connection.ChannelID,
			ChannelOwner: connection.ChannelOwner,
			ThingID:      connection.ThingID,
			ThingOwner:   connection.ThingOwner,
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

	for _, gtr := range req.GroupThingRelations {
		gRel := things.GroupThingRelation{
			ThingID:   gtr.ThingID,
			GroupID:   gtr.GroupID,
			CreatedAt: gtr.CreatedAt,
			UpdatedAt: gtr.UpdatedAt,
		}
		backup.GroupThingRelations = append(backup.GroupThingRelations, gRel)
	}

	return backup
}
