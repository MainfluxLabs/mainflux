package api

import (
	"context"

	"github.com/MainfluxLabs/mainflux/consumers"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/go-kit/kit/endpoint"
)

func restoreEndpoint(svc consumers.Consumer) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(restoreMessagesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := authorizeAdmin(ctx, req.token); err != nil {
			return nil, errors.Wrap(errors.ErrAuthorization, err)
		}

		if err := svc.Consume(req.Messages); err != nil {
			return nil, err
		}

		return restoreMessagesRes{}, nil
	}
}
