package backup

import (
	"context"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/go-kit/kit/endpoint"
)

func backupEndpoint(svc auth.Service) endpoint.Endpoint {
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

func restoreEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(restoreReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		backup := buildRestoreReq(req)

		err := svc.Restore(ctx, req.token, backup)
		if err != nil {
			return nil, err
		}

		return restoreRes{}, nil
	}
}

func buildBackupResponse(b auth.Backup) backupRes {
	res := backupRes{
		Orgs:           []viewOrgRes{},
		OrgMemberships: []viewOrgMembership{},
	}

	for _, org := range b.Orgs {
		view := viewOrgRes{
			ID:          org.ID,
			OwnerID:     org.OwnerID,
			Name:        org.Name,
			Description: org.Description,
			Metadata:    org.Metadata,
			CreatedAt:   org.CreatedAt,
			UpdatedAt:   org.UpdatedAt,
		}
		res.Orgs = append(res.Orgs, view)
	}

	for _, om := range b.OrgMemberships {
		view := viewOrgMembership{
			OrgID:     om.OrgID,
			MemberID:  om.MemberID,
			Role:      om.Role,
			CreatedAt: om.CreatedAt,
			UpdatedAt: om.UpdatedAt,
		}
		res.OrgMemberships = append(res.OrgMemberships, view)
	}

	return res
}

func buildRestoreReq(req restoreReq) (b auth.Backup) {
	for _, org := range req.Orgs {
		o := auth.Org{
			ID:          org.ID,
			OwnerID:     org.OwnerID,
			Name:        org.Name,
			Description: org.Description,
			Metadata:    org.Metadata,
			CreatedAt:   org.CreatedAt,
			UpdatedAt:   org.UpdatedAt,
		}
		b.Orgs = append(b.Orgs, o)
	}

	for _, om := range req.OrgMemberships {
		m := auth.OrgMembership{
			OrgID:     om.OrgID,
			MemberID:  om.MemberID,
			Role:      om.Role,
			CreatedAt: om.CreatedAt,
			UpdatedAt: om.UpdatedAt,
		}
		b.OrgMemberships = append(b.OrgMemberships, m)
	}

	return b
}
