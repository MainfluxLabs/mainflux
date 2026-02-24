// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package backup

import (
	"context"
	"encoding/base64"
	"encoding/json"

	"github.com/MainfluxLabs/mainflux/downlinks"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/cron"
	"github.com/go-kit/kit/endpoint"
)

func backupEndpoint(svc downlinks.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(backupReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		dls, err := svc.Backup(ctx, req.token)
		if err != nil {
			return nil, err
		}

		fileName := "downlinks-backup.json"
		return buildBackupResponse(dls, fileName)
	}
}

func restoreEndpoint(svc downlinks.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(restoreReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		dls := buildDownlinks(req)

		if err := svc.Restore(ctx, req.token, dls); err != nil {
			return nil, err
		}

		return restoreRes{}, nil
	}
}

func buildBackupResponse(dls []downlinks.Downlink, fileName string) (apiutil.ViewFileRes, error) {
	resp := backupResponse{
		Downlinks: make([]downlinkResponse, 0, len(dls)),
	}

	for _, dl := range dls {
		resp.Downlinks = append(resp.Downlinks, downlinkResponse{
			ID:      dl.ID,
			GroupID: dl.GroupID,
			ThingID: dl.ThingID,
			Name:    dl.Name,
			Url:     dl.Url,
			Method:  dl.Method,
			Payload: base64.StdEncoding.EncodeToString(dl.Payload),
			Headers: dl.Headers,
			Scheduler: schedulerRes{
				TimeZone:  dl.Scheduler.TimeZone,
				Frequency: dl.Scheduler.Frequency,
				DateTime:  dl.Scheduler.DateTime,
				Week: weekRes{
					Days: dl.Scheduler.Week.Days,
					Time: dl.Scheduler.Week.Time,
				},
				DayTime: dl.Scheduler.DayTime,
				Hour:    dl.Scheduler.Hour,
				Minute:  dl.Scheduler.Minute,
			},
			TimeFilter: timeFilterRes{
				StartParam: dl.TimeFilter.StartParam,
				EndParam:   dl.TimeFilter.EndParam,
				Format:     dl.TimeFilter.Format,
				Forecast:   dl.TimeFilter.Forecast,
				Interval:   dl.TimeFilter.Interval,
				Value:      dl.TimeFilter.Value,
			},
			Metadata: dl.Metadata,
		})
	}

	data, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return apiutil.ViewFileRes{}, err
	}

	return apiutil.ViewFileRes{
		File:     data,
		FileName: fileName,
	}, nil
}

func buildDownlinks(req restoreReq) []downlinks.Downlink {
	dls := make([]downlinks.Downlink, 0, len(req.Downlinks))

	for _, dlReq := range req.Downlinks {
		payload, err := base64.StdEncoding.DecodeString(dlReq.Payload)
		if err != nil {
			// If not base64, use raw string as payload
			payload = []byte(dlReq.Payload)
		}

		dls = append(dls, downlinks.Downlink{
			ID:      dlReq.ID,
			GroupID: dlReq.GroupID,
			ThingID: dlReq.ThingID,
			Name:    dlReq.Name,
			Url:     dlReq.Url,
			Method:  dlReq.Method,
			Payload: payload,
			Headers: dlReq.Headers,
			Scheduler: cron.Scheduler{
				TimeZone:  dlReq.Scheduler.TimeZone,
				Frequency: dlReq.Scheduler.Frequency,
				DateTime:  dlReq.Scheduler.DateTime,
				Week: cron.Week{
					Days: dlReq.Scheduler.Week.Days,
					Time: dlReq.Scheduler.Week.Time,
				},
				DayTime: dlReq.Scheduler.DayTime,
				Hour:    dlReq.Scheduler.Hour,
				Minute:  dlReq.Scheduler.Minute,
			},
			TimeFilter: downlinks.TimeFilter{
				StartParam: dlReq.TimeFilter.StartParam,
				EndParam:   dlReq.TimeFilter.EndParam,
				Format:     dlReq.TimeFilter.Format,
				Forecast:   dlReq.TimeFilter.Forecast,
				Interval:   dlReq.TimeFilter.Interval,
				Value:      dlReq.TimeFilter.Value,
			},
			Metadata: dlReq.Metadata,
		})
	}

	return dls
}
