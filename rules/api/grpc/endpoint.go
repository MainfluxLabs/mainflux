package grpc

import (
	"context"

	"github.com/MainfluxLabs/mainflux/rules"
	"github.com/go-kit/kit/endpoint"
)

func publishEndpoint(svc rules.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(publishReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.Publish(ctx, req.message); err != nil {
			return emptyRes{}, err
		}

		return emptyRes{}, nil
	}
}
