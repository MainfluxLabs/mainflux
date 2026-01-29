// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"

	"github.com/MainfluxLabs/mainflux/consumers/alarms"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/go-kit/kit/endpoint"
)

func listAlarmsByGroupEndpoint(svc alarms.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listAlarmsByGroupReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListAlarmsByGroup(ctx, req.token, req.groupID, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildAlarmsPageResponse(page, req.pageMetadata), nil
	}
}

func listAlarmsByThingEndpoint(svc alarms.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listAlarmsByThingReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListAlarmsByThing(ctx, req.token, req.thingID, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildAlarmsPageResponse(page, req.pageMetadata), nil
	}
}

func listAlarmsByOrgEndpoint(svc alarms.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(listAlarmsByOrgReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListAlarmsByOrg(ctx, req.token, req.orgID, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildAlarmsPageResponse(page, req.pageMetadata), nil
	}
}

func viewAlarmEndpoint(svc alarms.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(alarmReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		alarm, err := svc.ViewAlarm(ctx, req.token, req.id)
		if err != nil {
			return nil, err
		}

		return buildAlarmResponse(alarm), nil
	}
}

func removeAlarmsEndpoint(svc alarms.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(removeAlarmsReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.RemoveAlarms(ctx, req.token, req.AlarmIDs...); err != nil {
			return nil, err
		}

		return removeRes{}, nil
	}
}

func exportAlarmsByThingEndpoint(svc alarms.Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req := request.(exportAlarmsByThingReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ExportAlarmsByThing(ctx, req.token, req.thingID, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		var data []byte
		switch req.convertFormat {
		case jsonFormat:
			if data, err = ConvertToJSONFile(page, req.timeFormat); err != nil {
				return nil, errors.Wrap(errors.ErrBackupAlarms, err)
			}
		default:
			if data, err = ConvertToCSVFile(page, req.timeFormat); err != nil {
				return nil, errors.Wrap(errors.ErrBackupAlarms, err)
			}
		}

		return exportFileRes{
			file: data,
		}, nil
	}
}

func buildAlarmsPageResponse(ap alarms.AlarmsPage, pm apiutil.PageMetadata) AlarmsPageRes {
	res := AlarmsPageRes{
		Total:  ap.Total,
		Offset: pm.Offset,
		Limit:  pm.Limit,
		Order:  pm.Order,
		Dir:    pm.Dir,
		Alarms: []alarmResponse{},
	}

	for _, a := range ap.Alarms {
		res.Alarms = append(res.Alarms, buildAlarmResponse(a))
	}

	return res
}

func buildAlarmResponse(alarm alarms.Alarm) alarmResponse {
	return alarmResponse{
		ID:       alarm.ID,
		ThingID:  alarm.ThingID,
		GroupID:  alarm.GroupID,
		RuleID:   alarm.RuleID,
		Subtopic: alarm.Subtopic,
		Protocol: alarm.Protocol,
		Payload:  alarm.Payload,
		Created:  alarm.Created,
	}
}
