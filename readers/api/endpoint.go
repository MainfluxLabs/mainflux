// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"

	auth "github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
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

		return listMessagesRes{
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

		// Check if user is authorized to read all messages
		if err := authorizeAdmin(ctx, auth.RootSubject, req.token); err != nil {
			return nil, err
		}

		page, err := svc.ListAllMessages(req.pageMeta)
		if err != nil {
			return nil, err
		}

		return listMessagesRes{
			PageMetadata: page.PageMetadata,
			Total:        page.Total,
			Messages:     page.Messages,
		}, nil
	}
}

func restoreEndpoint(svc readers.MessageRepository) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(restoreMessagesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		// Check if user is authorized to read all messages
		if err := authorizeAdmin(ctx, auth.RootSubject, req.token); err != nil {
			return nil, err
		}

		messages := buildRestoreRequest(ctx, req)

		if err := svc.Restore(ctx, messages); err != nil {
			return nil, err
		}

		return restoreMessagesRes{}, nil
	}
}

func buildRestoreRequest(ctx context.Context, req restoreMessagesReq) []senml.Message {
	messages := make([]senml.Message, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = senml.Message{
			Channel:     msg.Channel,
			Subtopic:    msg.Subtopic,
			Publisher:   msg.Publisher,
			Protocol:    msg.Protocol,
			Name:        msg.Name,
			Unit:        msg.Unit,
			Time:        msg.Time,
			UpdateTime:  msg.UpdateTime,
			Value:       msg.Value,
			StringValue: msg.StringValue,
			DataValue:   msg.DataValue,
			BoolValue:   msg.BoolValue,
			Sum:         msg.Sum,
		}
	}

	return messages
}
