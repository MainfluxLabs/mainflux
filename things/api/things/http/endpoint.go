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

func listChannelsByThingEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listByConnectionReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListChannelsByThing(ctx, req.token, req.id, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		res := channelsPageRes{
			pageRes: pageRes{
				Total:  page.Total,
				Offset: page.Offset,
				Limit:  page.Limit,
			},
			Channels: []viewChannelRes{},
		}
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

func connectThingEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		cr := request.(connectThingReq)

		if err := cr.validate(); err != nil {
			return nil, err
		}

		if err := svc.Connect(ctx, cr.token, []string{cr.chanID}, []string{cr.thingID}); err != nil {
			return nil, err
		}

		return connectThingRes{}, nil
	}
}

func connectEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		cr := request.(connectReq)

		if err := cr.validate(); err != nil {
			return nil, err
		}

		if err := svc.Connect(ctx, cr.token, cr.ChannelIDs, cr.ThingIDs); err != nil {
			return nil, err
		}

		return connectRes{}, nil
	}
}

func disconnectEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		cr := request.(connectReq)
		if err := cr.validate(); err != nil {
			return nil, err
		}

		if err := svc.Disconnect(ctx, cr.token, cr.ChannelIDs, cr.ThingIDs); err != nil {
			return nil, err
		}

		return disconnectRes{}, nil
	}
}

func disconnectThingEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(connectThingReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.Disconnect(ctx, req.token, []string{req.chanID}, []string{req.thingID}); err != nil {
			return nil, err
		}

		return disconnectThingRes{}, nil
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

		return backupRes{
			Things:      backup.Things,
			Channels:    backup.Channels,
			Connections: backup.Connections,
		}, nil
	}
}

func restoreEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(restoreReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		backup := things.Backup{
			Things:      req.Things,
			Channels:    req.Channels,
			Connections: req.Connections,
		}

		if err := svc.Restore(ctx, req.token, backup); err != nil {
			return nil, err
		}

		return restoreRes{}, nil
	}
}

func createGroupEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createGroupReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		group := things.Group{
			Name:        req.Name,
			Description: req.Description,
			Metadata:    req.Metadata,
		}

		group, err := svc.CreateGroup(ctx, req.token, group)
		if err != nil {
			return nil, err
		}

		return groupRes{created: true, id: group.ID}, nil
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

		page, err := svc.ListGroups(ctx, req.token, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildGroupsResponse(page), nil
	}
}
func listMembersEndpoint(svc things.Service) endpoint.Endpoint {
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
		page, err := svc.ListMembers(ctx, req.token, req.id, pm)
		if err != nil {
			return nil, err
		}

		return buildUsersResponse(page), nil
	}
}

func listMemberships(svc things.Service) endpoint.Endpoint {
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

		page, err := svc.ListMemberships(ctx, req.token, req.id, pm)
		if err != nil {
			return nil, err
		}

		return buildGroupsResponse(page), nil
	}
}

func assignEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(assignReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.Assign(ctx, req.token, req.groupID, req.Members...); err != nil {
			return nil, err
		}

		return assignRes{}, nil
	}
}

func unassignEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(unassignReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.Unassign(ctx, req.token, req.groupID, req.Members...); err != nil {
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

func buildUsersResponse(mp things.MemberPage) memberPageRes {
	res := memberPageRes{
		pageRes: pageRes{
			Total:  mp.Total,
			Offset: mp.Offset,
			Limit:  mp.Limit,
			Name:   mp.Name,
		},
		MemberIDs: []string{},
	}

	res.MemberIDs = append(res.MemberIDs, mp.Members...)

	return res
}
