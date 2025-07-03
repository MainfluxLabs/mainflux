package memberships

import (
	"context"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/go-kit/kit/endpoint"
)

func createMembershipsEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(membershipsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.CreateMemberships(ctx, req.token, req.orgID, req.OrgMemberships...); err != nil {
			return nil, err
		}

		return createRes{}, nil
	}
}

func viewMembershipEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(membershipReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		mb, err := svc.ViewMembership(ctx, req.token, req.orgID, req.memberID)
		if err != nil {
			return nil, err

		}

		member := viewMembershipRes{
			ID:    mb.MemberID,
			Email: mb.Email,
			Role:  mb.Role,
		}

		return member, nil
	}
}

func listMembershipsByOrgEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listMembershipsReq)
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

		page, err := svc.ListMembershipsByOrg(ctx, req.token, req.id, pm)
		if err != nil {
			return nil, err
		}

		return buildMembershipsResponse(page), nil
	}
}

func updateMembershipsEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(membershipsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.UpdateMemberships(ctx, req.token, req.orgID, req.OrgMemberships...); err != nil {
			return nil, err
		}

		return createRes{}, nil
	}
}

func removeMembershipsEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(removeMembershipsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RemoveMemberships(ctx, req.token, req.orgID, req.MemberIDs...); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func buildMembershipsResponse(omp auth.OrgMembershipsPage) membershipPageRes {
	res := membershipPageRes{
		pageRes: pageRes{
			Total:  omp.Total,
			Offset: omp.Offset,
			Limit:  omp.Limit,
		},
		Memberships: []viewMembershipRes{},
	}

	for _, om := range omp.OrgMemberships {
		m := viewMembershipRes{
			ID:    om.MemberID,
			Email: om.Email,
			Role:  om.Role,
		}
		res.Memberships = append(res.Memberships, m)
	}

	return res
}
