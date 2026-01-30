// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package backup

import (
	"context"

	"github.com/MainfluxLabs/mainflux/readers"
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

		return buildBackupResponse(backup), nil
	}
}

func restoreEndpoint(svc readers.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(restoreReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		// var (
		// 	messages     []readers.Message
		// 	jsonMessages []mfjson.Message
		// 	// err          error
		// )

		// switch req.fileType {
		// case jsonFormat:
		// 	if jsonMessages, err = ConvertJSONToJSONMessages(req.Messages); err != nil {
		// 		return nil, errors.Wrap(errors.ErrRestoreMessages, err)
		// 	}
		// default:
		// 	if jsonMessages, err = ConvertCSVToJSONMessages(req.Messages); err != nil {
		// 		return nil, errors.Wrap(errors.ErrRestoreMessages, err)
		// 	}
		// }

		// for _, msg := range jsonMessages {
		// 	messages = append(messages, msg)
		// }

		// if err := svc.RestoreJSONMessages(ctx, req.token, messages...); err != nil {
		// 	return nil, err
		// }

		return restoreMessagesRes{}, nil
	}
}

// func restoreSenMLMessagesEndpoint(svc readers.Service) endpoint.Endpoint {
// 	return func(ctx context.Context, request any) (any, error) {
// 		req := request.(restoreMessagesReq)
// 		if err := req.validate(); err != nil {
// 			return nil, err
// 		}
//
// var (
// 	messages      []readers.Message
// 	senmlMessages []senml.Message
// 	err           error
// )

// switch req.fileType {
// case jsonFormat:
// 	if senmlMessages, err = ConvertJSONToSenMLMessages(req.Messages); err != nil {
// 		return nil, errors.Wrap(errors.ErrRestoreMessages, err)
// 	}
// default:
// 	if senmlMessages, err = ConvertCSVToSenMLMessages(req.Messages); err != nil {
// 		return nil, errors.Wrap(errors.ErrRestoreMessages, err)
// 	}
// }

// for _, msg := range senmlMessages {
// 	messages = append(messages, msg)
// }

// if err := svc.RestoreSenMLMessages(ctx, req.token, messages...); err != nil {
// 	return nil, err
// }

//		return restoreMessagesRes{}, nil
//	}
//
// }

func buildBackupResponse(backup readers.Backup) backupRes {
	res := backupRes{
		JSONMessages:  backup.JSONMessages.Messages,
		SenMLMessages: backup.SenMLMessages.Messages,
	}
	return res
}
