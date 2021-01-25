//
// Copyright (c) 2019
// Mainflux
//
// SPDX-License-Identifier: Apache-2.0
//

package http

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/re"
)

func pingEndpoint(svc re.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(pingReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		greeting, err := svc.Ping(req.Secret)
		if err != nil {
			return nil, err
		}

		res := pingRes{
			Greeting: greeting,
		}
		return res, nil
	}
}
