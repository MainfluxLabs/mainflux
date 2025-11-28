package memberships

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/go-kit/kit/endpoint"
)

func createOrgMembershipsEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(orgMembershipsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.CreateOrgMemberships(ctx, req.token, req.orgID, req.OrgMemberships...); err != nil {
			return nil, err
		}

		return createRes{}, nil
	}
}

func viewOrgMembershipEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(orgMembershipReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		mb, err := svc.ViewOrgMembership(ctx, req.token, req.orgID, req.memberID)
		if err != nil {
			return nil, err

		}

		om := viewOrgMembershipRes{
			MemberID: mb.MemberID,
			Email:    mb.Email,
			Role:     mb.Role,
		}

		return om, nil
	}
}

func listOrgMembershipsEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listOrgMembershipsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		pm := apiutil.PageMetadata{
			Offset: req.offset,
			Limit:  req.limit,
			Email:  req.email,
			Order:  req.order,
			Dir:    req.dir,
		}

		page, err := svc.ListOrgMemberships(ctx, req.token, req.orgID, pm)
		if err != nil {
			return nil, err
		}

		return buildOrgMembershipsResponse(page, pm), nil
	}
}

func updateOrgMembershipsEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(orgMembershipsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.UpdateOrgMemberships(ctx, req.token, req.orgID, req.OrgMemberships...); err != nil {
			return nil, err
		}

		return createRes{}, nil
	}
}

func removeOrgMembershipsEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(removeOrgMembershipsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RemoveOrgMemberships(ctx, req.token, req.orgID, req.MemberIDs...); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func backupOrgMembershipsEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(backupByOrgReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		backup, err := svc.BackupOrgMemberships(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		fileName := fmt.Sprintf("org-memberships-backup-%s.json", req.id)
		return buildBackupResponse(backup, fileName)
	}
}

func restoreOrgMembershipsEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(restoreByOrgReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		orgMembershipsBackup := buildOrgMembershipsBackup(req.OrgMemberships)

		if err := svc.RestoreOrgMemberships(ctx, req.token, req.id, orgMembershipsBackup); err != nil {
			return nil, err
		}

		return restoreRes{}, nil
	}
}

func buildOrgMembershipsResponse(omp auth.OrgMembershipsPage, pm apiutil.PageMetadata) orgMembershipPageRes {
	res := orgMembershipPageRes{
		pageRes: pageRes{
			Total:  omp.Total,
			Offset: pm.Offset,
			Limit:  pm.Limit,
			Email:  pm.Email,
			Order:  pm.Order,
			Dir:    pm.Dir,
		},
		OrgMemberships: []viewOrgMembershipRes{},
	}

	for _, om := range omp.OrgMemberships {
		m := viewOrgMembershipRes{
			MemberID:  om.MemberID,
			OrgID:     om.OrgID,
			Email:     om.Email,
			Role:      om.Role,
			CreatedAt: om.CreatedAt,
			UpdatedAt: om.UpdatedAt,
		}
		res.OrgMemberships = append(res.OrgMemberships, m)
	}

	return res
}

func buildOrgMembershipsBackup(orgMemberships []viewOrgMembershipRes) (backup auth.OrgMembershipsBackup) {
	for _, membership := range orgMemberships {
		om := auth.OrgMembership{
			MemberID:  membership.MemberID,
			OrgID:     membership.OrgID,
			Email:     membership.Email,
			Role:      membership.Role,
			CreatedAt: membership.CreatedAt,
			UpdatedAt: membership.UpdatedAt,
		}
		backup.OrgMemberships = append(backup.OrgMemberships, om)
	}
	return backup
}

func buildBackupResponse(b auth.OrgMembershipsBackup, fileName string) (apiutil.ViewFileRes, error) {
	views := make([]viewOrgMembershipRes, 0, len(b.OrgMemberships))
	for _, membership := range b.OrgMemberships {
		views = append(views, viewOrgMembershipRes{
			MemberID:  membership.MemberID,
			OrgID:     membership.OrgID,
			Email:     membership.Email,
			Role:      membership.Role,
			CreatedAt: membership.CreatedAt,
			UpdatedAt: membership.UpdatedAt,
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
