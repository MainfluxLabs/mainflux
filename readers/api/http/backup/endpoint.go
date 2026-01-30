// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package backup

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/MainfluxLabs/mainflux/readers/api/http/messages"
	"github.com/go-kit/kit/endpoint"
)

func backupEndpoint(svc readers.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(backupReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		backup, err := svc.Backup(ctx, req.token)
		if err != nil {
			return nil, err
		}
		var json []byte
		if json, err = messages.ConvertJSONToJSONFile(backup.JSONMessages, ""); err != nil {
			return nil, errors.Wrap(errors.ErrBackupMessages, err)
		}
		var senml []byte
		if senml, err = messages.ConvertSenMLToJSONFile(backup.SenMLMessages, ""); err != nil {
			return nil, errors.Wrap(errors.ErrBackupMessages, err)
		}

		return backupFileRes{
			JSONMessages:  json,
			SenMLMessages: senml,
		}, nil
	}
}

func restoreEndpoint(svc readers.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(restoreReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		backup, err := buildRestoreReq(req)
		if err != nil {
			return nil, errors.Wrap(errors.ErrRestoreMessages, err)
		}

		if err := svc.Restore(ctx, req.token, backup); err != nil {
			return nil, err
		}

		return restoreRes{}, nil
	}
}

func buildRestoreReq(req restoreReq) (readers.Backup, error) {
	var backup readers.Backup

	if len(req.JSONMessages) > 0 {
		jsonMsgs, err := messages.ConvertJSONToJSONMessages(req.JSONMessages)
		if err != nil {
			return backup, err
		}

		backup.JSONMessages.Messages = make([]readers.Message, len(jsonMsgs))
		for i, m := range jsonMsgs {
			backup.JSONMessages.Messages[i] = m
		}
	}

	if len(req.SenMLMessages) > 0 {
		senmlMsgs, err := messages.ConvertJSONToSenMLMessages(req.SenMLMessages)
		if err != nil {
			return backup, err
		}

		backup.SenMLMessages.Messages = make([]readers.Message, len(senmlMsgs))
		for i, m := range senmlMsgs {
			backup.SenMLMessages.Messages[i] = m
		}
	}

	return backup, nil
}
