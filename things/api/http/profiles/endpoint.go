// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package profiles

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/go-kit/kit/endpoint"
)

func createProfilesEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
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
				Metadata: c.Metadata,
			}
			prs = append(prs, pr)
		}

		saved, err := svc.CreateProfiles(ctx, req.token, req.groupID, prs...)
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
	return func(ctx context.Context, request any) (any, error) {
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
	return func(ctx context.Context, request any) (any, error) {
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
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListProfiles(ctx, req.token, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildProfilesResponse(page, req.pageMetadata), nil
	}
}

func listProfilesByGroupEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listByGroupReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListProfilesByGroup(ctx, req.token, req.id, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildProfilesResponse(page, req.pageMetadata), nil
	}
}

func listProfilesByOrgEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listByOrgReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListProfilesByOrg(ctx, req.token, req.id, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildProfilesResponse(page, req.pageMetadata), nil
	}
}

func backupProfilesByOrgEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(backupByOrgReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		backup, err := svc.BackupProfilesByOrg(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		fileName := fmt.Sprintf("profiles-backup-by-org-%s.json", req.id)
		return buildBackupResponse(backup, fileName)
	}
}

func restoreProfilesByOrgEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(restoreByOrgReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		profilesBackup := buildProfilesBackup(req.Profiles)

		if err := svc.RestoreProfilesByOrg(ctx, req.token, req.id, profilesBackup); err != nil {
			return nil, err
		}

		return restoreRes{}, nil
	}
}

func backupProfilesByGroupEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(backupByGroupReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		backup, err := svc.BackupProfilesByGroup(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		fileName := fmt.Sprintf("profiles-backup-by-group-%s.json", req.id)
		return buildBackupResponse(backup, fileName)
	}
}

func restoreProfilesByGroupEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(restoreByGroupReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		profilesBackup := buildProfilesBackup(req.Profiles)

		if err := svc.RestoreProfilesByGroup(ctx, req.token, req.id, profilesBackup); err != nil {
			return nil, err
		}

		return restoreRes{}, nil
	}
}

func updateProfileEndpoint(svc things.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
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
	return func(ctx context.Context, request any) (any, error) {
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
	return func(ctx context.Context, request any) (any, error) {
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

func buildProfilesResponse(pp things.ProfilesPage, pm apiutil.PageMetadata) profilesPageRes {
	res := profilesPageRes{
		pageRes: pageRes{
			Total:  pp.Total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
			Order:  pm.Order,
			Dir:    pm.Dir,
			Name:   pm.Name,
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

func buildProfilesBackup(profiles []viewProfileRes) (backup things.ProfilesBackup) {
	for _, profile := range profiles {
		pr := things.Profile{
			ID:       profile.ID,
			GroupID:  profile.GroupID,
			Name:     profile.Name,
			Config:   profile.Config,
			Metadata: profile.Metadata,
		}
		backup.Profiles = append(backup.Profiles, pr)
	}
	return backup
}

func buildBackupResponse(pb things.ProfilesBackup, fileName string) (apiutil.ViewFileRes, error) {
	views := make([]viewProfileRes, 0, len(pb.Profiles))
	for _, profile := range pb.Profiles {
		views = append(views, viewProfileRes{
			ID:       profile.ID,
			GroupID:  profile.GroupID,
			Name:     profile.Name,
			Metadata: profile.Metadata,
			Config:   profile.Config,
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
