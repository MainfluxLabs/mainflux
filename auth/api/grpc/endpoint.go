// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/go-kit/kit/endpoint"
)

func issueEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(issueReq)
		if err := req.validate(); err != nil {
			return issueRes{}, err
		}

		key := auth.Key{
			Type:     req.keyType,
			Subject:  req.email,
			IssuerID: req.id,
			IssuedAt: time.Now().UTC(),
		}

		_, secret, err := svc.Issue(ctx, "", key)
		if err != nil {
			return issueRes{}, err
		}

		return issueRes{secret}, nil
	}
}

func identifyEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(identityReq)
		if err := req.validate(); err != nil {
			return identityRes{}, err
		}

		id, err := svc.Identify(ctx, req.token)
		if err != nil {
			return identityRes{}, err
		}

		ret := identityRes{
			id:    id.ID,
			email: id.Email,
		}

		return ret, nil
	}
}

func authorizeEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(authReq)

		if err := req.validate(); err != nil {
			return emptyRes{}, err
		}

		ar := auth.AuthzReq{
			Token:   req.Token,
			Object:  req.Object,
			Subject: req.Subject,
			Action:  req.Action,
		}

		if err := svc.Authorize(ctx, ar); err != nil {
			return emptyRes{}, err
		}

		return emptyRes{}, nil
	}
}

func assignRoleEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(assignRoleReq)

		if err := req.validate(); err != nil {
			return emptyRes{}, err
		}

		if err := svc.AssignRole(ctx, req.ID, req.Role); err != nil {
			return emptyRes{}, err
		}

		return emptyRes{}, nil
	}
}

func addPolicyEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(policyReq)
		if err := req.validate(); err != nil {
			return emptyRes{}, err
		}

		if err := svc.AddPolicy(ctx, req.Token, req.Object, req.Policy); err != nil {
			return emptyRes{}, err
		}

		return emptyRes{}, nil
	}
}

func assignEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(assignReq)

		if err := req.validate(); err != nil {
			return emptyRes{}, err
		}

		_, err := svc.Identify(ctx, req.token)
		if err != nil {
			return emptyRes{}, err
		}

		if err := svc.AssignGroups(ctx, req.token, req.groupID, req.memberID); err != nil {
			return emptyRes{}, err
		}

		return emptyRes{}, nil
	}
}

func membersEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(membersReq)
		if err := req.validate(); err != nil {
			return orgMembersRes{}, err
		}

		pm := auth.PageMetadata{
			Offset: req.offset,
			Limit:  req.limit,
		}
		omp, err := svc.ListOrgMembers(ctx, req.token, req.groupID, pm)
		if err != nil {
			return orgMembersRes{}, err
		}
		var omIDs []string
		for _, id := range omp.OrgMembers {
			omIDs = append(omIDs, id.MemberID)
		}
		return orgMembersRes{
			orgMemberIDs: omIDs,
			offset:       req.offset,
			limit:        req.limit,
			total:        omp.PageMetadata.Total,
		}, nil
	}
}
