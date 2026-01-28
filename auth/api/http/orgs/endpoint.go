package orgs

import (
	"context"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/go-kit/kit/endpoint"
)

func createOrgsEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
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
	return func(ctx context.Context, request any) (any, error) {
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
	return func(ctx context.Context, request any) (any, error) {
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
	return func(ctx context.Context, request any) (any, error) {
		req := request.(orgReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RemoveOrgs(ctx, req.token, req.id); err != nil {
			return nil, err
		}

		return deleteRes{}, nil
	}
}

func deleteOrgsEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(deleteOrgsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RemoveOrgs(ctx, req.token, req.OrgIDs...); err != nil {
			return nil, err
		}

		return deleteRes{}, nil
	}
}

func listOrgsEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listOrgsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListOrgs(ctx, req.token, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildOrgsResponse(page, req.pageMetadata), nil
	}
}

func buildOrgsResponse(op auth.OrgsPage, pm apiutil.PageMetadata) orgsPageRes {
	res := orgsPageRes{
		pageRes: pageRes{
			Total:  op.Total,
			Limit:  pm.Limit,
			Offset: pm.Offset,
			Ord:    pm.Order,
			Dir:    pm.Dir,
			Name:   pm.Name,
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
