// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/go-kit/kit/endpoint"
)

func listUsersByIDsEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(getUsersByIDsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		up, err := svc.ListUsersByIDs(ctx, req.ids, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		res := getUsersRes{
			pageMetadata: &protomfx.PageMetadata{
				Total:  up.Total,
				Offset: req.pageMetadata.Offset,
				Limit:  req.pageMetadata.Limit,
				Email:  req.pageMetadata.Email,
				Order:  req.pageMetadata.Order,
				Dir:    req.pageMetadata.Dir,
			},
		}

		for _, u := range up.Users {
			user := protomfx.User{
				Id:     u.ID,
				Email:  u.Email,
				Status: u.Status,
			}
			res.users = append(res.users, &user)
		}

		return res, nil
	}
}

func listUsersByEmailsEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(getUsersByEmailsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		users, err := svc.ListUsersByEmails(ctx, req.emails)
		if err != nil {
			return nil, err
		}

		mu := []*protomfx.User{}

		for _, u := range users {
			user := protomfx.User{
				Id:     u.ID,
				Email:  u.Email,
				Status: u.Status,
			}
			mu = append(mu, &user)
		}

		return getUsersRes{users: mu}, nil
	}
}
