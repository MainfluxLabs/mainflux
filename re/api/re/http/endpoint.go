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

func infoEndpoint(svc re.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		info, err := svc.Info()
		if err != nil {
			return nil, err
		}
		return infoRes{
			Version:       info.Version,
			Os:            info.Os,
			UpTimeSeconds: info.UpTimeSeconds,
		}, nil
	}
}

func listEndpoint(svc re.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		streams, err := svc.List()
		if err != nil {
			return nil, err
		}
		return listRes{
			Streams: streams,
		}, nil
	}
}

func createStreamEndpoint(svc re.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(streamReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		result, err := svc.CreateStream(req.SQL)
		if err != nil {
			return nil, err
		}

		return createRes{
			Result: result,
		}, nil
	}
}

func viewEndpoint(svc re.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		stream, err := svc.View(req.id)
		if err != nil {
			return nil, err
		}
		return viewRes{
			Stream: stream,
		}, nil
	}
}
