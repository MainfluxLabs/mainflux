// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"net/http"
	"time"

	"github.com/MainfluxLabs/mainflux/ota"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/go-kit/kit/endpoint"
)

func createUpdateEndpoint(svc ota.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(createUpdateReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		u := ota.Update{
			FileName:    req.FileName,
			Version:     req.Version,
			FileSize:    req.FileSize,
			Checksum:    req.Checksum,
			Description: req.Description,
			Timeout:     time.Duration(req.Timeout) * time.Second,
		}

		saved, err := svc.CreateUpdate(ctx, req.token, req.groupID, u, req.ThingIDs...)
		if err != nil {
			return nil, err
		}

		return buildUpdateRes(saved, true), nil
	}
}

func listUpdatesEndpoint(svc ota.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listUpdatesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListUpdatesByGroup(ctx, req.token, req.groupID, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildUpdatesPageRes(page, req.pageMetadata), nil
	}
}

func viewUpdateEndpoint(svc ota.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(updateReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		u, err := svc.ViewUpdate(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		return buildUpdateRes(u, false), nil
	}
}

func removeUpdateEndpoint(svc ota.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(updateReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RemoveUpdate(ctx, req.token, req.id); err != nil {
			return nil, err
		}

		return apiutil.EmptyRes{StatusCode: http.StatusNoContent}, nil
	}
}

func listThingUpdatesEndpoint(svc ota.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listThingUpdatesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListThingUpdates(ctx, req.token, req.updateID, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildThingUpdatesPageRes(page, req.pageMetadata), nil
	}
}

func retryFailedEndpoint(svc ota.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(updateReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RetryFailed(ctx, req.token, req.id); err != nil {
			return nil, err
		}

		return apiutil.EmptyRes{StatusCode: http.StatusOK}, nil
	}
}

func viewPendingEndpoint(svc ota.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(viewPendingReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		cmd, err := svc.ViewPending(ctx, req.key)
		switch {
		case errors.Contains(err, dbutil.ErrNotFound):
			return apiutil.EmptyRes{StatusCode: http.StatusNoContent}, nil
		case err != nil:
			return nil, err
		}

		return buildPendingRes(cmd), nil
	}
}

func reportStatusEndpoint(svc ota.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(reportStatusReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		report := ota.StatusReport{
			UpdateID: req.UpdateID,
			Status:   req.Status,
			Version:  req.Version,
			Error:    req.Error,
		}

		if err := svc.ReportStatus(ctx, req.key, report); err != nil {
			return nil, err
		}

		return apiutil.EmptyRes{StatusCode: http.StatusOK}, nil
	}
}
