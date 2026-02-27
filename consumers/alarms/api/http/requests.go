package http

import (
	"github.com/MainfluxLabs/mainflux/consumers/alarms"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

const (
	minLen       = 1
	maxLimitSize = 200
	maxNameSize  = 254
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

	return apiutil.ValidatePageMetadata(req.pageMetadata, maxLimitSize, maxNameSize, alarms.AllowedOrders)
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

	return apiutil.ValidatePageMetadata(req.pageMetadata, maxLimitSize, maxNameSize, alarms.AllowedOrders)
}

type listAlarmsByOrgReq struct {
	token        string
	orgID        string
	pageMetadata apiutil.PageMetadata
}

func (req listAlarmsByOrgReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.orgID == "" {
		return apiutil.ErrMissingOrgID
	}

	return apiutil.ValidatePageMetadata(req.pageMetadata, maxLimitSize, maxNameSize, alarms.AllowedOrders)
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

type exportAlarmsByThingReq struct {
	token         string
	thingID       string
	convertFormat string
	timeFormat    string
	pageMetadata  apiutil.PageMetadata
}

func (req exportAlarmsByThingReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.thingID == "" {
		return apiutil.ErrMissingThingID
	}

	if req.convertFormat != jsonFormat && req.convertFormat != csvFormat {
		return apiutil.ErrInvalidQueryParams
	}

	return apiutil.ValidatePageMetadata(req.pageMetadata, maxLimitSize, maxNameSize, alarms.AllowedOrders)
}
