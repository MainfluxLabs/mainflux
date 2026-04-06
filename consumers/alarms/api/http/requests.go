package http

import (
	"slices"

	"github.com/MainfluxLabs/mainflux/consumers/alarms"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

const (
	minLen       = 1
	maxLimitSize = 200
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
	pageMetadata alarms.PageMetadata
}

func (req listAlarmsByGroupReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.groupID == "" {
		return apiutil.ErrMissingGroupID
	}

	return req.pageMetadata.Validate(maxLimitSize)
}

type listAlarmsByThingReq struct {
	token        string
	thingID      string
	pageMetadata alarms.PageMetadata
}

func (req listAlarmsByThingReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.thingID == "" {
		return apiutil.ErrMissingThingID
	}

	return req.pageMetadata.Validate(maxLimitSize)
}

type listAlarmsByOrgReq struct {
	token        string
	orgID        string
	pageMetadata alarms.PageMetadata
}

func (req listAlarmsByOrgReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.orgID == "" {
		return apiutil.ErrMissingOrgID
	}

	return req.pageMetadata.Validate(maxLimitSize)
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

	if slices.Contains(req.AlarmIDs, "") {
		return apiutil.ErrMissingAlarmID
	}

	return nil
}

type updateAlarmStatusReq struct {
	token  string
	id     string
	Status string `json:"status"`
}

func (req updateAlarmStatusReq) validate() error {
	if req.token == "" {
		return apiutil.ErrBearerToken
	}

	if req.id == "" {
		return apiutil.ErrMissingAlarmID
	}

	switch req.Status {
	case alarms.AlarmStatusActive, alarms.AlarmStatusNoted, alarms.AlarmStatusCleared:
	default:
		return apiutil.ErrInvalidStatus
	}

	return nil
}

type exportAlarmsByThingReq struct {
	token         string
	thingID       string
	convertFormat string
	timeFormat    string
	pageMetadata  alarms.PageMetadata
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

	return req.pageMetadata.Validate(maxLimitSize)
}
