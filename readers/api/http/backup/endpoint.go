// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package backup

import (
	"context"
	"encoding/json"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
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

		backup, err := buildRestoreReq(req)
		if err != nil {
			return nil, errors.Wrap(apiutil.ErrMalformedEntity, err)
		}

		err = svc.Restore(ctx, req.token, backup)
		if err != nil {
			return nil, err
		}

		return restoreRes{}, nil
	}
}

func buildBackupResponse(backup readers.Backup) backupRes {
	res := backupRes{
		JSONMessages:  backup.JSONMessages.Messages,
		SenMLMessages: backup.SenMLMessages.Messages,
	}
	return res
}

func buildRestoreReq(req restoreReq) (readers.Backup, error) {
	var backup readers.Backup

	if err := json.Unmarshal(req.Messages, &backup); err != nil {
		return readers.Backup{}, err
	}

	return backup, nil
}
