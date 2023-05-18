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
			return authorizeRes{}, err
		}

		err := svc.AuthorizeAdmin(ctx, auth.AuthzReq{Email: req.Email})
		if err != nil {
			return authorizeRes{}, err
		}
		return authorizeRes{authorized: true}, err
	}
}

func accessGroupEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(accessGroupReq)
		if err := req.validate(); err != nil {
			return emptyRes{}, err
		}

		if err := svc.CanAccessGroup(ctx, req.Token, req.GroupID); err != nil {
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

		err = svc.AssignMembersByIDs(ctx, req.token, req.memberID, req.groupID)
		if err != nil {
			return emptyRes{}, err
		}
		return emptyRes{}, nil

	}
}

func membersEndpoint(svc auth.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(membersReq)
		if err := req.validate(); err != nil {
			return membersRes{}, err
		}

		pm := auth.PageMetadata{
			Offset: req.offset,
			Limit:  req.limit,
		}
		mp, err := svc.ListOrgMembers(ctx, req.token, req.groupID, pm)
		if err != nil {
			return membersRes{}, err
		}
		var members []string
		for _, id := range mp.Members {
			members = append(members, id.ID)
		}
		return membersRes{
			offset:  req.offset,
			limit:   req.limit,
			total:   mp.PageMetadata.Total,
			members: members,
		}, nil
	}
}
