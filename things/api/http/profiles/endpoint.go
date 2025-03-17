// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package profiles

import (
	"context"

	"github.com/MainfluxLabs/mainflux/things"
	"github.com/go-kit/kit/endpoint"
)

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

func viewProfileByThingEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewByThingReq)

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

func listProfilesEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listReq)

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

func listProfilesByGroupEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listByGroupReq)
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

func listProfilesByOrgEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listByOrgReq)
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
