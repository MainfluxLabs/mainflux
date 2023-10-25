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

		page, err := svc.ListOrgs(ctx, req.token, req.admin, pm)
		if err != nil {
			return nil, err
		}

		return buildOrgsResponse(page), nil
	}
}

func listMemberships(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listOrgMembershipsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		pm := auth.PageMetadata{
			Name:     req.name,
			Offset:   req.offset,
			Limit:    req.limit,
			Metadata: req.metadata,
		}

		page, err := svc.ListOrgMemberships(ctx, req.token, req.id, pm)
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

func listMembersEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listOrgMembersReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		pm := auth.PageMetadata{
			Offset:   req.offset,
			Limit:    req.limit,
			Metadata: req.metadata,
		}
		page, err := svc.ListOrgMembers(ctx, req.token, req.id, pm)
		if err != nil {
			return nil, err
		}

		return buildMembersResponse(page), nil
	}
}

func assignOrgGroupsEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(groupsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.AssignGroups(ctx, req.token, req.orgID, req.GroupIDs...); err != nil {
			return nil, err
		}

		return assignRes{}, nil
	}
}

func unassignOrgGroupsEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(groupsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.UnassignGroups(ctx, req.token, req.orgID, req.GroupIDs...); err != nil {
			return nil, err
		}

		return unassignRes{}, nil
	}
}

func listGroupsEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listOrgGroupsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		pm := auth.PageMetadata{
			Offset:   req.offset,
			Limit:    req.limit,
			Metadata: req.metadata,
		}
		page, err := svc.ListOrgGroups(ctx, req.token, req.id, pm)
		if err != nil {
			return nil, err
		}

		return buildGroupsResponse(page), nil
	}
}

func createGroupPoliciesEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(groupPoliciesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		var gpByEmails []auth.GroupPolicyByEmail
		for _, g := range req.GroupPolicies {
			gpByEmail := auth.GroupPolicyByEmail{
				Email:  g.Email,
				Policy: g.Policy,
			}
			gpByEmails = append(gpByEmails, gpByEmail)
		}

		if err := svc.CreateGroupPolicies(ctx, req.token, req.groupID, gpByEmails...); err != nil {
			return nil, err
		}

		return createGroupPoliciesRes{}, nil
	}
}

func updateGroupPoliciesEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(groupPoliciesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		var gpByEmails []auth.GroupPolicyByEmail
		for _, g := range req.GroupPolicies {
			giByEmail := auth.GroupPolicyByEmail{
				Email:  g.Email,
				Policy: g.Policy,
			}

			gpByEmails = append(gpByEmails, giByEmail)
		}

		if err := svc.UpdateGroupPolicies(ctx, req.token, req.groupID, gpByEmails...); err != nil {
			return nil, err
		}

		return updateGroupPoliciesRes{}, nil
	}
}

func removeGroupMembersEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(removeGroupPoliciesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RemoveGroupPolicies(ctx, req.token, req.groupID, req.MemberIDs...); err != nil {
			return nil, err
		}

		return deleteRes{}, nil
	}
}

func listGroupPoliciesEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listGroupMembersReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		pm := auth.PageMetadata{
			Offset: req.offset,
			Limit:  req.limit,
		}

		gpp, err := svc.ListGroupPolicies(ctx, req.token, req.groupID, pm)
		if err != nil {
			return nil, err
		}

		return buildGroupPoliciesResponse(gpp), nil
	}
}

func viewGroupMembershipEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewGroupMembershipReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		o, err := svc.ViewGroupMembership(ctx, req.token, req.groupID)
		if err != nil {
			return nil, err
		}

		org := viewOrgRes{
			ID:          o.ID,
			Name:        o.Name,
			OwnerID:     o.OwnerID,
			Description: o.Description,
			Metadata:    o.Metadata,
			CreatedAt:   o.CreatedAt,
			UpdatedAt:   o.UpdatedAt,
		}

		return org, nil
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

		backup := auth.Backup{
			Orgs:       req.Orgs,
			OrgMembers: req.OrgMembers,
			OrgGroups:  req.OrgGroups,
		}

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

func buildGroupsResponse(gp auth.GroupsPage) groupsPageRes {
	res := groupsPageRes{
		pageRes: pageRes{
			Total:  gp.Total,
			Offset: gp.Offset,
			Limit:  gp.Limit,
		},
		Groups: []viewGroupRes{},
	}

	for _, group := range gp.Groups {
		g := viewGroupRes{
			ID:          group.ID,
			OwnerID:     group.OwnerID,
			Name:        group.Name,
			Description: group.Description,
		}
		res.Groups = append(res.Groups, g)
	}

	return res
}

func buildBackupResponse(b auth.Backup) backupRes {
	res := backupRes{
		Orgs:       []viewOrgRes{},
		OrgMembers: []viewOrgMembers{},
		OrgGroups:  []viewOrgGroups{},
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

	for _, groupRel := range b.OrgGroups {
		view := viewOrgGroups{
			GroupID:   groupRel.GroupID,
			OrgID:     groupRel.OrgID,
			CreatedAt: groupRel.CreatedAt,
			UpdatedAt: groupRel.UpdatedAt,
		}
		res.OrgGroups = append(res.OrgGroups, view)
	}

	return res
}

func buildGroupPoliciesResponse(gpp auth.GroupPoliciesPage) listGroupPoliciesRes {
	res := listGroupPoliciesRes{
		pageRes: pageRes{
			Total:  gpp.Total,
			Limit:  gpp.Limit,
			Offset: gpp.Offset,
		},
		GroupPolicies: []groupPolicy{},
	}

	for _, g := range gpp.GroupMembersPolicies {
		gp := groupPolicy{
			Email:  g.Email,
			ID:     g.MemberID,
			Policy: g.Policy,
		}
		res.GroupPolicies = append(res.GroupPolicies, gp)
	}

	return res
}
