// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"strings"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	mfjson "github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/go-kit/kit/endpoint"
)

func listAllMessagesEndpoint(svc readers.MessageRepository) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listAllMessagesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		var page readers.MessagesPage
		switch {
		case req.key != "":
			pc, err := getPubConfByKey(ctx, req.key)
			if err != nil {
				return nil, err
			}
			req.pageMeta.Publisher = pc.PublisherID
			req.pageMeta.Format = dbutil.GetTableName(pc.ProfileConfig.GetContentType())

			p, err := svc.ListAllMessages(req.pageMeta)
			if err != nil {
				return nil, err
			}

			page = p
		default:
			// Check if user is authorized to read all messages
			if err := isAdmin(ctx, req.token); err != nil {
				return nil, err
			}

			p, err := svc.ListAllMessages(req.pageMeta)
			if err != nil {
				return nil, err
			}

			page = p
		}

		return listMessagesRes{
			PageMetadata: page.PageMetadata,
			Total:        page.Total,
			Messages:     page.Messages,
		}, nil
	}
}

func deleteMessagesEndpoint(svc readers.MessageRepository) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteMessagesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		var table string

		switch {
		case req.key != "":
			pc, err := getPubConfByKey(ctx, req.key)
			if err != nil {
				return nil, errors.Wrap(errors.ErrAuthentication, err)
			}

			req.pageMeta.Publisher = pc.PublisherID
			table = dbutil.GetTableName(pc.ProfileConfig.GetContentType())

		case req.token != "":
			if err := isAdmin(ctx, req.token); err != nil {
				return nil, err
			}
		default:
			return nil, errors.ErrAuthentication
		}

		err := svc.DeleteMessages(ctx, req.pageMeta, table)
		if err != nil {
			return nil, err
		}

		return nil, nil
	}
}

func backupMessagesEndpoint(svc readers.MessageRepository) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(backupMessagesReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := isAdmin(ctx, req.token); err != nil {
			return nil, err
		}

		req.pageMeta.Format = dbutil.GetTableName(req.messageFormat)
		page, err := svc.Backup(req.pageMeta)
		if err != nil {
			return nil, err
		}

		var data []byte
		outputFormat := strings.ToLower(strings.TrimSpace(req.convertFormat))
		switch outputFormat {
		case jsonFormat:
			data, err = apiutil.GenerateJSON(page)
		case csvFormat:
			data, err = apiutil.GenerateCSV(page, req.pageMeta.Format)
		default:
			return nil, errors.Wrap(errors.ErrMalformedEntity, err)
		}

		if err != nil {
			return nil, errors.Wrap(errors.ErrBackupMessages, err)
		}

		return backupFileRes{
			file: data,
		}, nil

	}
}

func restoreMessagesEndpoint(svc readers.MessageRepository) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(restoreMessagesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := isAdmin(ctx, req.token); err != nil {
			return nil, err
		}

		var messages []readers.Message
		var jsonMessages []mfjson.Message
		var senmlMessages []senml.Message
		var err error

		switch req.messageFormat {
		case messaging.JSONFormat:
			switch req.fileType {
			case jsonFormat:
				jsonMessages, err = apiutil.ConvertJSONToJSONMessages(req.Messages)
			case csvFormat:
				jsonMessages, err = apiutil.ConvertCSVToJSONMessages(req.Messages)
			default:
				return nil, errors.Wrap(errors.ErrMessage, err)
			}

			for _, msg := range jsonMessages {
				messages = append(messages, msg)
			}

		case messaging.SenMLFormat:
			switch req.fileType {
			case jsonFormat:
				senmlMessages, err = apiutil.ConvertJSONToSenMLMessages(req.Messages)
			case csvFormat:
				senmlMessages, err = apiutil.ConvertCSVToSenMLMessages(req.Messages)
			default:
				return nil, errors.Wrap(errors.ErrMessage, err)
			}

			for _, msg := range senmlMessages {
				messages = append(messages, msg)
			}
		default:
			return nil, errors.Wrap(errors.ErrRestoreMessages, err)
		}

		table := dbutil.GetTableName(req.messageFormat)
		if err := svc.Restore(ctx, table, messages...); err != nil {
			return nil, err
		}

		return restoreMessagesRes{}, nil
	}
}
