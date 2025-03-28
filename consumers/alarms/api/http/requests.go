package http

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

const (
	minLen       = 1
	maxLimitSize = 100
)

type alarmReq struct {
	token string
	id    string
}

func (req *alarmReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.id == "" {
		return apiutil.ErrMissingAlarmID
	}
	return nil
}

type listAlarmsByGroupReq struct {
	token        string
	groupID      string
	pageMetadata apiutil.PageMetadata
}

func (req listAlarmsByGroupReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.groupID == "" {
		return apiutil.ErrMissingGroupID
	}
	return apiutil.ValidatePageMetadata(req.pageMetadata, maxLimitSize, 0)
}

type listAlarmsByThingReq struct {
	token        string
	thingID      string
	pageMetadata apiutil.PageMetadata
}

func (req listAlarmsByThingReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if req.thingID == "" {
		return apiutil.ErrMissingThingID
	}
	return apiutil.ValidatePageMetadata(req.pageMetadata, maxLimitSize, 0)
}

type removeAlarmsReq struct {
	token    string
	AlarmIDs []string `json:"alarm_ids,omitempty"`
}

func (req removeAlarmsReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}
	if len(req.AlarmIDs) < minLen {
		return apiutil.ErrEmptyList
	}
	for _, alarmID := range req.AlarmIDs {
		if alarmID == "" {
			return apiutil.ErrMissingAlarmID
		}
	}
	return nil
}
