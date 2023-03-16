// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc

import (
	"context"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/go-kit/kit/endpoint"
)

func listUsersByIDsEndpoint(svc users.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getUsersByIDsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		up, err := svc.ListUsersByIDs(ctx, req.IDs)
		if err != nil {
			return nil, err
		}

		mu := []*mainflux.User{}

		for _, u := range up.Users {
			user := mainflux.User{
				Id:     u.ID,
				Email:  u.Email,
				Status: u.Status,
			}
			mu = append(mu, &user)
		}

		return getUsersByIDsRes{Users: mu}, nil
	}
}
