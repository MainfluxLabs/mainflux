// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	mfjson "github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/go-kit/kit/endpoint"
)

func listJSONMessagesEndpoint(svc readers.MessageRepository) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listJSONMessagesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		var page readers.JSONMessagesPage
		pc, err := getPubConfByKey(ctx, req.key)
		if err != nil {
			return nil, err
		}
		req.pageMeta.Publisher = pc.PublisherID

		page, err = svc.ListJSONMessages(ctx, req.pageMeta)
		if err != nil {
			return nil, err
		}

		return listJSONMessagesRes{
			JSONPageMetadata: req.pageMeta,
			Total:            page.Total,
			Messages:         page.Messages,
		}, nil
	}
}

func listAdminJSONMessagesEndpoint(svc readers.MessageRepository) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listJSONMessagesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := isAdmin(ctx, req.token); err != nil {
			return nil, err
		}

		if req.pageMeta.Publisher == "" {
			return nil, errors.New("publisherID is required for admin query")
		}

		page, err := svc.ListJSONMessages(ctx, req.pageMeta)
		if err != nil {
			return nil, err
		}

		return listJSONMessagesRes{
			JSONPageMetadata: req.pageMeta,
			Total:            page.Total,
			Messages:         page.Messages,
		}, nil
	}
}

func listSenMLMessagesEndpoint(svc readers.MessageRepository) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listSenMLMessagesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		pc, err := getPubConfByKey(ctx, req.key)
		if err != nil {
			return nil, err
		}
		req.pageMeta.Publisher = pc.PublisherID

		page, err := svc.ListSenMLMessages(ctx, req.pageMeta)
		if err != nil {
			return nil, err
		}

		return listSenMLMessagesRes{
			SenMLPageMetadata: req.pageMeta,
			Total:             page.Total,
			Messages:          page.Messages,
		}, nil
	}
}

func listAdminSenMLMessagesEndpoint(svc readers.MessageRepository) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listSenMLMessagesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := isAdmin(ctx, req.token); err != nil {
			return nil, err
		}

		if req.pageMeta.Publisher == "" {
			return nil, errors.New("publisherID is required for admin query")
		}

		page, err := svc.ListSenMLMessages(ctx, req.pageMeta)
		if err != nil {
			return nil, err
		}

		return listSenMLMessagesRes{
			SenMLPageMetadata: req.pageMeta,
			Total:             page.Total,
			Messages:          page.Messages,
		}, nil
	}
}

func deleteJSONMessagesEndpoint(svc readers.MessageRepository) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteJSONMessagesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		switch {
		case req.key != "":
			pc, err := getPubConfByKey(ctx, req.key)
			if err != nil {
				return nil, errors.Wrap(errors.ErrAuthentication, err)
			}
			req.pageMeta.Publisher = pc.PublisherID
		case req.token != "":
			if err := isAdmin(ctx, req.token); err != nil {
				return nil, err
			}
		default:
			return nil, errors.ErrAuthentication
		}

		err := svc.DeleteJSONMessages(ctx, req.pageMeta)
		if err != nil {
			return nil, err
		}

		return nil, nil
	}
}

func deleteSenMLMessagesEndpoint(svc readers.MessageRepository) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteSenMLMessagesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		switch {
		case req.key != "":
			pc, err := getPubConfByKey(ctx, req.key)
			if err != nil {
				return nil, errors.Wrap(errors.ErrAuthentication, err)
			}
			req.pageMeta.Publisher = pc.PublisherID

		case req.token != "":
			if err := isAdmin(ctx, req.token); err != nil {
				return nil, err
			}
		default:
			return nil, errors.ErrAuthentication
		}

		err := svc.DeleteSenMLMessages(ctx, req.pageMeta)
		if err != nil {
			return nil, err
		}

		return nil, nil
	}
}

func backupJSONMessagesEndpoint(svc readers.MessageRepository) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(backupJSONMessagesReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := isAdmin(ctx, req.token); err != nil {
			return nil, err
		}

		page, err := svc.BackupJSONMessages(ctx, req.pageMeta)
		if err != nil {
			return nil, err
		}

		var data []byte
		switch req.convertFormat {
		case jsonFormat:
			if data, err = apiutil.GenerateJSON(page.MessagesPage); err != nil {
				return nil, errors.Wrap(errors.ErrBackupMessages, err)
			}
		default:
			if data, err = apiutil.GenerateCSVFromJSON(page.MessagesPage); err != nil {
				return nil, errors.Wrap(errors.ErrBackupMessages, err)
			}
		}

		return backupFileRes{
			file: data,
		}, nil
	}
}

func backupSenMLMessagesEndpoint(svc readers.MessageRepository) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(backupSenMLMessagesReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := isAdmin(ctx, req.token); err != nil {
			return nil, err
		}

		page, err := svc.BackupSenMLMessages(ctx, req.pageMeta)
		if err != nil {
			return nil, err
		}

		var data []byte
		switch req.convertFormat {
		case jsonFormat:
			if data, err = apiutil.GenerateJSON(page.MessagesPage); err != nil {
				return nil, errors.Wrap(errors.ErrBackupMessages, err)
			}
		default:
			if data, err = apiutil.GenerateCSVFromSenML(page.MessagesPage); err != nil {
				return nil, errors.Wrap(errors.ErrBackupMessages, err)
			}
		}

		return backupFileRes{
			file: data,
		}, nil
	}
}

func restoreJSONMessagesEndpoint(svc readers.MessageRepository) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(restoreMessagesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := isAdmin(ctx, req.token); err != nil {
			return nil, err
		}

		var (
			messages     []readers.Message
			jsonMessages []mfjson.Message
			err          error
		)

		switch req.fileType {
		case jsonFormat:
			if jsonMessages, err = apiutil.ConvertJSONToJSONMessages(req.Messages); err != nil {
				return nil, errors.Wrap(errors.ErrRestoreMessages, err)
			}
		default:
			if jsonMessages, err = apiutil.ConvertCSVToJSONMessages(req.Messages); err != nil {
				return nil, errors.Wrap(errors.ErrRestoreMessages, err)
			}
		}

		for _, msg := range jsonMessages {
			messages = append(messages, msg)
		}

		if err := svc.RestoreJSONMessages(ctx, messages...); err != nil {
			return nil, err
		}

		return restoreMessagesRes{}, nil
	}
}

func restoreSenMLMessagesEndpoint(svc readers.MessageRepository) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(restoreMessagesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := isAdmin(ctx, req.token); err != nil {
			return nil, err
		}

		var (
			messages      []readers.Message
			senmlMessages []senml.Message
			err           error
		)

		switch req.fileType {
		case jsonFormat:
			if senmlMessages, err = apiutil.ConvertJSONToSenMLMessages(req.Messages); err != nil {
				return nil, errors.Wrap(errors.ErrRestoreMessages, err)
			}
		default:
			if senmlMessages, err = apiutil.ConvertCSVToSenMLMessages(req.Messages); err != nil {
				return nil, errors.Wrap(errors.ErrRestoreMessages, err)
			}
		}

		for _, msg := range senmlMessages {
			messages = append(messages, msg)
		}

		if err := svc.RestoreSenMLMessages(ctx, messages...); err != nil {
			return nil, err
		}

		return restoreMessagesRes{}, nil
	}
}
