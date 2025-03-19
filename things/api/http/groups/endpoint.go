// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package groups

import (
	"context"

	"github.com/MainfluxLabs/mainflux/things"
	"github.com/go-kit/kit/endpoint"
)

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

func viewGroupByThingEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewByThingReq)
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

func viewGroupByProfileEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewByProfileReq)
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

func listGroupsEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listReq)
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
		req := request.(listByOrgReq)
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
