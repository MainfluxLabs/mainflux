package members

import (
	"context"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/go-kit/kit/endpoint"
)

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
