// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package backup

import (
	"context"
	"encoding/json"

	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/go-kit/kit/endpoint"
)

func backupMessagesEndpoint(svc readers.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(backupMessagesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		backup, err := svc.Backup(ctx, req.token)
		if err != nil {
			return nil, err
		}

		data, err := buildMessagesBackupResponse(backup)
		if err != nil {
			return nil, err
		}

		return backupFileRes{
			file: data,
		}, nil
	}
}

func restoreMessagesEndpoint(svc readers.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(restoreMessagesReq)
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

// Add senml/json name
func buildMessagesBackupResponse(backup readers.Backup) ([]byte, error) {
	allMsgs := make([]any, 0, len(backup.JSONMessages.Messages)+len(backup.SenMLMessages.Messages))

	for _, m := range backup.JSONMessages.Messages {
		allMsgs = append(allMsgs, m)
	}

	for _, m := range backup.SenMLMessages.Messages {
		allMsgs = append(allMsgs, m)
	}

	return json.Marshal(allMsgs)
}
