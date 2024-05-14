package orgs

import (
	"context"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/go-kit/kit/endpoint"
)

func createOrgsEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createOrgsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		org := auth.Org{
			Name:        req.Name,
			Description: req.Description,
			Metadata:    req.Metadata,
		}

		org, err := svc.CreateOrg(ctx, req.token, org)
		if err != nil {
			return nil, err
		}

		return orgRes{created: true, id: org.ID}, nil
	}
}

func viewOrgEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(orgReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		org, err := svc.ViewOrg(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		res := viewOrgRes{
			ID:          org.ID,
			Name:        org.Name,
			Description: org.Description,
			Metadata:    org.Metadata,
			OwnerID:     org.OwnerID,
			CreatedAt:   org.CreatedAt,
			UpdatedAt:   org.UpdatedAt,
		}

		return res, nil
	}
}

func updateOrgEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateOrgReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		org := auth.Org{
			ID:          req.id,
			Name:        req.Name,
			Description: req.Description,
			Metadata:    req.Metadata,
		}

		_, err := svc.UpdateOrg(ctx, req.token, org)
		if err != nil {
			return nil, err
		}

		return orgRes{created: false}, nil
	}
}

func deleteOrgEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(orgReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RemoveOrg(ctx, req.token, req.id); err != nil {
			return nil, err
		}

		return deleteRes{}, nil
	}
}

func listOrgsEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listOrgsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		pm := auth.PageMetadata{
			Name:     req.name,
			Metadata: req.metadata,
			Offset:   req.offset,
			Limit:    req.limit,
		}

		page, err := svc.ListOrgs(ctx, req.token, pm)
		if err != nil {
			return nil, err
		}

		return buildOrgsResponse(page), nil
	}
}

func listOrgsByMemberEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listOrgsByMemberReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		pm := auth.PageMetadata{
			Name:     req.name,
			Offset:   req.offset,
			Limit:    req.limit,
			Metadata: req.metadata,
		}

		page, err := svc.ListOrgsByMember(ctx, req.token, req.id, pm)
		if err != nil {
			return nil, err
		}

		return buildOrgsResponse(page), nil
	}
}

func viewMemberEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(memberReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		mb, err := svc.ViewMember(ctx, req.token, req.orgID, req.memberID)
		if err != nil {
			return nil, err

		}

		member := viewMemberRes{
			ID:    mb.MemberID,
			Email: mb.Email,
			Role:  mb.Role,
		}

		return member, nil
	}
}

func assignMembersEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(membersReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.AssignMembers(ctx, req.token, req.orgID, req.OrgMembers...); err != nil {
			return nil, err
		}

		return assignRes{}, nil
	}
}

func unassignMembersEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(unassignMembersReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.UnassignMembers(ctx, req.token, req.orgID, req.MemberIDs...); err != nil {
			return nil, err
		}

		return unassignRes{}, nil
	}
}

func updateMembersEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(membersReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.UpdateMembers(ctx, req.token, req.orgID, req.OrgMembers...); err != nil {
			return nil, err
		}

		return assignRes{}, nil
	}
}

func listMembersByOrgEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listMembersByOrgReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		pm := auth.PageMetadata{
			Offset:   req.offset,
			Limit:    req.limit,
			Metadata: req.metadata,
		}
		page, err := svc.ListMembersByOrg(ctx, req.token, req.id, pm)
		if err != nil {
			return nil, err
		}

		return buildMembersResponse(page), nil
	}
}

func backupEndpoint(svc auth.Service) endpoint.Endpoint {
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

func restoreEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
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

func buildOrgsResponse(op auth.OrgsPage) orgsPageRes {
	res := orgsPageRes{
		pageRes: pageRes{
			Total:  op.Total,
			Limit:  op.Limit,
			Offset: op.Offset,
		},
		Orgs: []viewOrgRes{},
	}

	for _, org := range op.Orgs {
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

	return res
}

func buildMembersResponse(omp auth.OrgMembersPage) memberPageRes {
	res := memberPageRes{
		pageRes: pageRes{
			Total:  omp.Total,
			Offset: omp.Offset,
			Limit:  omp.Limit,
		},
		Members: []viewMemberRes{},
	}

	for _, memb := range omp.OrgMembers {
		m := viewMemberRes{
			ID:    memb.MemberID,
			Email: memb.Email,
			Role:  memb.Role,
		}
		res.Members = append(res.Members, m)
	}

	return res
}

func buildBackupResponse(b auth.Backup) backupRes {
	res := backupRes{
		Orgs:       []viewOrgRes{},
		OrgMembers: []viewOrgMembers{},
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

	for _, mRel := range b.OrgMembers {
		view := viewOrgMembers{
			OrgID:     mRel.OrgID,
			MemberID:  mRel.MemberID,
			Role:      mRel.Role,
			CreatedAt: mRel.CreatedAt,
			UpdatedAt: mRel.UpdatedAt,
		}
		res.OrgMembers = append(res.OrgMembers, view)
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

	for _, om := range req.OrgMembers {
		m := auth.OrgMember{
			OrgID:     om.OrgID,
			MemberID:  om.MemberID,
			Role:      om.Role,
			CreatedAt: om.CreatedAt,
			UpdatedAt: om.UpdatedAt,
		}
		b.OrgMembers = append(b.OrgMembers, m)
	}

	return b
}
