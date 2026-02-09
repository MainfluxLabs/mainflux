// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package messages

import (
	"context"
	"sync"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/readers"
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

func searchMessagesEndpoint(svc readers.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(searchMessagesReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		results := make([]searchResultItem, len(req.Searches))

		var (
			wg       sync.WaitGroup
			mu       sync.Mutex
			firstErr error
		)

		for i, search := range req.Searches {
			wg.Add(1)
			go func(idx int, s searchRequest) {
				defer wg.Done()

				item := searchResultItem{Type: s.Type}

				switch s.Type {
				case "json":
					page, err := svc.ListJSONMessages(ctx, req.token, req.thingKey, *s.JSON)
					if err != nil {
						item.Error = err.Error()
						mu.Lock()
						if firstErr == nil {
							firstErr = err
						}
						mu.Unlock()
					} else {
						item.Total = page.Total
						item.Messages = page.Messages
					}
				case "senml":
					page, err := svc.ListSenMLMessages(ctx, req.token, req.thingKey, *s.SenML)
					if err != nil {
						item.Error = err.Error()
						mu.Lock()
						if firstErr == nil {
							firstErr = err
						}
						mu.Unlock()
					} else {
						item.Total = page.Total
						item.Messages = page.Messages
					}
				}
				mu.Lock()
				results[idx] = item
				mu.Unlock()
			}(i, search)
		}

		wg.Wait()
		return searchMessagesRes{Results: results}, nil
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
