// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"strings"

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
		switch {
		case req.key != "":
			pc, err := getPubConfByKey(ctx, req.key)
			if err != nil {
				return nil, err
			}
			req.pageMeta.Publisher = pc.PublisherID
		default:
			if err := isAdmin(ctx, req.token); err != nil {
				return nil, err
			}
		}

		page, err := svc.ListJSONMessages(req.pageMeta)
		if err != nil {
			return nil, err
		}

		return listJSONMessagesRes{
			JSONMetadata: page.JSONMetadata,
			Total:        page.Total,
			Messages:     page.Messages,
		}, nil
	}
}

func listSenMLMessagesEndpoint(svc readers.MessageRepository) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listSenMLMessagesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		var page readers.SenMLMessagesPage
		switch {
		case req.key != "":
			pc, err := getPubConfByKey(ctx, req.key)
			if err != nil {
				return nil, err
			}
			req.pageMeta.Publisher = pc.PublisherID
		default:
			if err := isAdmin(ctx, req.token); err != nil {
				return nil, err
			}
		}

		page, err := svc.ListSenMLMessages(req.pageMeta)
		if err != nil {
			return nil, err
		}

		return listSenMLMessagesRes{
			SenMLMetadata: page.SenMLMetadata,
			Total:         page.Total,
			Messages:      page.Messages,
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

		req.pageMeta.Format = defFormat
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

		page, err := svc.BackupJSONMessages(req.pageMeta)
		if err != nil {
			return nil, err
		}

		var data []byte
		outputFormat := strings.ToLower(strings.TrimSpace(req.convertFormat))
		switch outputFormat {
		case jsonFormat:
			data, err = apiutil.GenerateJSON(page)
		case csvFormat:
			data, err = apiutil.GenerateCSV(page, "json")
		default:
			return nil, errors.Wrap(errors.ErrBackupMessages, err)
		}

		if err != nil {
			return nil, errors.Wrap(errors.ErrBackupMessages, err)
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

		page, err := svc.BackupSenMLMessages(req.pageMeta)
		if err != nil {
			return nil, err
		}

		var data []byte
		outputFormat := strings.ToLower(strings.TrimSpace(req.convertFormat))
		switch outputFormat {
		case jsonFormat:
			if data, err = apiutil.GenerateJSON(page); err != nil {
				return nil, errors.Wrap(errors.ErrBackupMessages, err)
			}
		case csvFormat:
			if data, err = apiutil.GenerateCSV(page, "senml"); err != nil {
				return nil, errors.Wrap(errors.ErrBackupMessages, err)
			}
		default:
			return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
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
		case csvFormat:
			if jsonMessages, err = apiutil.ConvertCSVToJSONMessages(req.Messages); err != nil {
				return nil, errors.Wrap(errors.ErrRestoreMessages, err)
			}
		default:
			return nil, errors.Wrap(errors.ErrRestoreMessages, err)
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
		case csvFormat:
			if senmlMessages, err = apiutil.ConvertCSVToSenMLMessages(req.Messages); err != nil {
				return nil, errors.Wrap(errors.ErrRestoreMessages, err)
			}
		default:
			return nil, errors.Wrap(errors.ErrRestoreMessages, err)
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
