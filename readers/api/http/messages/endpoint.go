// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package messages

import (
	"context"
	"sync"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/go-kit/kit/endpoint"
)

func listJSONMessagesEndpoint(svc readers.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listJSONMessagesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListJSONMessages(ctx, req.token, req.thingKey, req.pageMeta)
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

func listSenMLMessagesEndpoint(svc readers.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listSenMLMessagesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListSenMLMessages(ctx, req.token, req.thingKey, req.pageMeta)
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

func searchJSONMessagesEndpoint(svc readers.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(searchJSONMessagesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		results := make([]searchJSONResultItem, len(req.Searches))
		sem := make(chan struct{}, apiutil.ConcurrencyLimit)
		var wg sync.WaitGroup

		for i, search := range req.Searches {
			wg.Add(1)
			sem <- struct{}{}
			go func(idx int, pm readers.JSONPageMetadata) {
				defer wg.Done()
				defer func() { <-sem }()

				var item searchJSONResultItem
				if page, err := svc.ListJSONMessages(ctx, req.token, things.ThingKey{}, pm); err != nil {
					item.Error = err.Error()
				} else {
					item.Total = page.Total
					item.Messages = page.Messages
				}
				results[idx] = item
			}(i, search)
		}

		wg.Wait()
		return searchJSONMessagesRes(results), nil
	}
}

func searchSenMLMessagesEndpoint(svc readers.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(searchSenMLMessagesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		results := make([]searchSenMLResultItem, len(req.Searches))
		sem := make(chan struct{}, apiutil.ConcurrencyLimit)
		var wg sync.WaitGroup

		for i, search := range req.Searches {
			wg.Add(1)
			sem <- struct{}{}
			go func(idx int, pm readers.SenMLPageMetadata) {
				defer wg.Done()
				defer func() { <-sem }()

				var item searchSenMLResultItem
				if page, err := svc.ListSenMLMessages(ctx, req.token, things.ThingKey{}, pm); err != nil {
					item.Error = err.Error()
				} else {
					item.Total = page.Total
					item.Messages = page.Messages
				}
				results[idx] = item
			}(i, search)
		}

		wg.Wait()
		return searchSenMLMessagesRes(results), nil
	}
}

func deleteAllJSONMessagesEndpoint(svc readers.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(deleteAllJSONMessagesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.DeleteAllJSONMessages(ctx, req.token, req.pageMeta); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func deleteJSONMessagesEndpoint(svc readers.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(deleteJSONMessagesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.DeleteJSONMessages(ctx, req.token, req.pageMeta); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func deleteAllSenMLMessagesEndpoint(svc readers.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(deleteAllSenMLMessagesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.DeleteAllSenMLMessages(ctx, req.token, req.pageMeta); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func deleteSenMLMessagesEndpoint(svc readers.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(deleteSenMLMessagesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.DeleteSenMLMessages(ctx, req.token, req.pageMeta); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func exportJSONMessagesEndpoint(svc readers.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(exportJSONMessagesReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ExportJSONMessages(ctx, req.token, req.pageMeta)
		if err != nil {
			return nil, err
		}

		var data []byte
		switch req.convertFormat {
		case jsonFormat:
			if data, err = ConvertJSONToJSONFile(page, req.timeFormat); err != nil {
				return nil, errors.Wrap(errors.ErrBackupMessages, err)
			}
		default:
			if data, err = ConvertJSONToCSVFile(page, req.timeFormat); err != nil {
				return nil, errors.Wrap(errors.ErrBackupMessages, err)
			}
		}

		return exportFileRes{
			file: data,
		}, nil
	}
}

func exportSenMLMessagesEndpoint(svc readers.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(exportSenMLMessagesReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ExportSenMLMessages(ctx, req.token, req.pageMeta)
		if err != nil {
			return nil, err
		}

		var data []byte
		switch req.convertFormat {
		case jsonFormat:
			if data, err = ConvertSenMLToJSONFile(page, req.timeFormat); err != nil {
				return nil, errors.Wrap(errors.ErrBackupMessages, err)
			}
		default:
			if data, err = ConvertSenMLToCSVFile(page, req.timeFormat); err != nil {
				return nil, errors.Wrap(errors.ErrBackupMessages, err)
			}
		}

		return exportFileRes{
			file: data,
		}, nil
	}
}
