package orgs

import (
	"context"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/go-kit/kit/endpoint"
)

func createOrgEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createOrgReq)
		if err := req.validate(); err != nil {
			return orgRes{}, err
		}

		org := auth.Org{
			Name:        req.Name,
			Description: req.Description,
			Metadata:    req.Metadata,
		}

		org, err := svc.CreateOrg(ctx, req.token, org)
		if err != nil {
			return orgRes{}, err
		}

		return orgRes{created: true, id: org.ID}, nil
	}
}

func viewOrgEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(orgReq)
		if err := req.validate(); err != nil {
			return viewOrgRes{}, err
		}

		org, err := svc.ViewOrg(ctx, req.token, req.id)
		if err != nil {
			return viewOrgRes{}, err
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
			return orgRes{}, err
		}

		org := auth.Org{
			ID:          req.id,
			Name:        req.Name,
			Description: req.Description,
			Metadata:    req.Metadata,
		}

		_, err := svc.UpdateOrg(ctx, req.token, org)
		if err != nil {
			return orgRes{}, err
		}

		res := orgRes{created: false}
		return res, nil
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
			return orgPageRes{}, err
		}
		pm := auth.OrgPageMetadata{
			Metadata: req.metadata,
		}
		page, err := svc.ListOrgs(ctx, req.token, pm)
		if err != nil {
			return orgPageRes{}, err
		}

		return buildOrgsResponse(page), nil
	}
}

func listMemberships(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listOrgMembershipsReq)
		if err := req.validate(); err != nil {
			return memberPageRes{}, err
		}

		pm := auth.OrgPageMetadata{
			Offset:   req.offset,
			Limit:    req.limit,
			Metadata: req.metadata,
		}

		page, err := svc.ListOrgMemberships(ctx, req.token, req.id, pm)
		if err != nil {
			return memberPageRes{}, err
		}

		return buildOrgsResponse(page), nil
	}
}

func shareOrgAccessEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(shareOrgAccessReq)
		if err := req.validate(); err != nil {
			return shareOrgRes{}, err
		}

		if err := svc.AssignOrgAccessRights(ctx, req.token, req.ThingOrgID, req.userOrgID); err != nil {
			return shareOrgRes{}, err
		}
		return shareOrgRes{}, nil
	}
}

func assignEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(assignOrgReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.AssignOrg(ctx, req.token, req.orgID, req.Members...); err != nil {
			return nil, err
		}

		return assignRes{}, nil
	}
}

func unassignEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(unassignOrgReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.UnassignOrg(ctx, req.token, req.orgID, req.Members...); err != nil {
			return nil, err
		}

		return unassignRes{}, nil
	}
}

func listMembersEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listOrgMembersReq)
		if err := req.validate(); err != nil {
			return memberPageRes{}, err
		}

		pm := auth.OrgPageMetadata{
			Offset:   req.offset,
			Limit:    req.limit,
			Metadata: req.metadata,
		}
		page, err := svc.ListOrgMembers(ctx, req.token, req.id, pm)
		if err != nil {
			return memberPageRes{}, err
		}

		return buildUsersResponse(page), nil
	}
}

func toViewOrgRes(org auth.Org) viewOrgRes {
	view := viewOrgRes{
		ID:          org.ID,
		OwnerID:     org.OwnerID,
		Name:        org.Name,
		Description: org.Description,
		Metadata:    org.Metadata,
		CreatedAt:   org.CreatedAt,
		UpdatedAt:   org.UpdatedAt,
	}

	return view
}

func buildOrgsResponse(gp auth.OrgPage) orgPageRes {
	res := orgPageRes{
		pageRes: pageRes{
			Total: gp.Total,
		},
		Orgs: []viewOrgRes{},
	}

	for _, org := range gp.Orgs {
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

func buildUsersResponse(mp auth.OrgMembersPage) memberPageRes {
	res := memberPageRes{
		pageRes: pageRes{
			Total:  mp.Total,
			Offset: mp.Offset,
			Limit:  mp.Limit,
			Name:   mp.Name,
		},
		Members: []string{},
	}

	for _, m := range mp.Members {
		res.Members = append(res.Members, m.ID)
	}

	return res
}
