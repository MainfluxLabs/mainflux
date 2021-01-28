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

func createEndpoint(svc re.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(streamReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		result, err := svc.CreateStream(req.Name, req.Topic, req.Row)
		if err != nil {
			return nil, err
		}

		return streamRes{
			Result: result,
		}, nil
	}
}

func updateEndpoint(svc re.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(streamReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		result, err := svc.UpdateStream(req.Name, req.Topic, req.Row)
		if err != nil {
			return nil, err
		}

		return streamRes{
			Result: result,
		}, nil
	}
}

func listEndpoint(svc re.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		streams, err := svc.ListStreams()
		if err != nil {
			return nil, err
		}
		return listStreamsRes{
			Streams: streams,
		}, nil
	}
}

func viewEndpoint(svc re.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewStreamReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		stream, err := svc.ViewStream(req.name)
		if err != nil {
			return nil, err
		}
		return viewStreamRes{
			Stream: stream,
		}, nil
	}
}

func deleteEndpoint(svc re.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(viewStreamReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		result, err := svc.DeleteStream(req.name)
		if err != nil {
			return nil, err
		}

		return streamRes{
			Result: result,
		}, nil
	}
}
