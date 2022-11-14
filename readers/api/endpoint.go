// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/go-kit/kit/endpoint"
)

func ListChannelMessagesEndpoint(svc readers.MessageRepository) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listChannelMessagesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := authorize(ctx, req.token, req.key, req.chanID); err != nil {
			return nil, errors.Wrap(errors.ErrAuthorization, err)
		}

		page, err := svc.ListChannelMessages(req.chanID, req.pageMeta)
		if err != nil {
			return nil, err
		}

		return listChannelMessagesPageRes{
			PageMetadata: page.PageMetadata,
			Total:        page.Total,
			Messages:     page.Messages,
		}, nil
	}
}

func listAllMessagesEndpoint(svc readers.MessageRepository) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listAllMessagesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		user, err := identify(ctx, req.token)
		if err != nil {
			return nil, err
		}
		// Check if user is authorized to read all messages
		if err := authorizeUser(ctx, user.id, "authorities", "member"); err != nil {
			return nil, errors.Wrap(errors.ErrAuthentication, err)
		}

		msgs, err := svc.ListAllMessages()
		if err != nil {
			return nil, err
		}

		return listAllMessagesRes{
			Messages: msgs,
		}, nil
	}
}

type userIdentity struct {
	id    string
	email string
}

func identify(ctx context.Context, token string) (userIdentity, error) {
	var auth mainflux.AuthServiceClient
	identity, err := auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return userIdentity{}, errors.Wrap(errors.ErrAuthentication, err)
	}

	return userIdentity{identity.Id, identity.Email}, nil
}

func authorizeUser(ctx context.Context, subject, object, relation string) error {
	req := &mainflux.AuthorizeReq{
		Sub: subject,
		Obj: object,
		Act: relation,
	}
	var auth mainflux.AuthServiceClient
	res, err := auth.Authorize(ctx, req)
	if err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}
	if !res.GetAuthorized() {
		return errors.ErrAuthorization
	}

	return nil
}
