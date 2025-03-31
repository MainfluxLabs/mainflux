// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"

	"github.com/MainfluxLabs/mainflux/consumers/alarms"
	"github.com/go-kit/kit/endpoint"
)

func listAlarmsByGroupEndpoint(svc alarms.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listAlarmsByGroupReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListAlarmsByGroup(ctx, req.token, req.groupID, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildAlarmsResponse(page), nil
	}
}

func listAlarmsByThingEndpoint(svc alarms.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listAlarmsByThingReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		page, err := svc.ListAlarmsByThing(ctx, req.token, req.thingID, req.pageMetadata)
		if err != nil {
			return nil, err
		}

		return buildAlarmsResponse(page), nil
	}
}

func viewAlarmEndpoint(svc alarms.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
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
	return func(ctx context.Context, request interface{}) (interface{}, error) {
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

func buildAlarmsResponse(ap alarms.AlarmsPage) AlarmsPageRes {
	res := AlarmsPageRes{
		Total:  ap.Total,
		Offset: ap.Offset,
		Limit:  ap.Limit,
		Alarms: []alarmResponse{},
	}

	for _, a := range ap.Alarms {
		alarm := alarmResponse{
			ID:       a.ID,
			ThingID:  a.ThingID,
			GroupID:  a.GroupID,
			Subtopic: a.Subtopic,
			Protocol: a.Protocol,
			Rule:     a.Rule,
			Payload:  a.Payload,
			Created:  a.Created,
		}
		res.Alarms = append(res.Alarms, alarm)
	}

	return res
}

func buildAlarmResponse(alarm alarms.Alarm) alarmResponse {
	return alarmResponse{
		ID:       alarm.ID,
		ThingID:  alarm.ThingID,
		GroupID:  alarm.GroupID,
		Subtopic: alarm.Subtopic,
		Protocol: alarm.Protocol,
		Rule:     alarm.Rule,
		Payload:  alarm.Payload,
		Created:  alarm.Created,
	}
}
