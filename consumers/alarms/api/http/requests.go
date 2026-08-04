package http

import (
	"slices"

	"github.com/MainfluxLabs/mainflux/consumers/alarms"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

const (
	minLen        = 1
	maxLimitSize  = 200
	minAlarmLevel = 1
	maxAlarmLevel = 5
)

// validatePageMetadata validates the alarms page metadata.
func validatePageMetadata(pm alarms.PageMetadata) error {
	common := apiutil.PageMetadata{Offset: pm.Offset, Limit: pm.Limit, Order: pm.Order, Dir: pm.Dir}
	if err := common.Validate(maxLimitSize, alarms.AlarmOrderFields); err != nil {
		return err
	}

	if pm.Level != 0 && (pm.Level < minAlarmLevel || pm.Level > maxAlarmLevel) {
		return apiutil.ErrInvalidAlarmLevel
	}

	if pm.Status != "" {
		switch pm.Status {
		case alarms.StatusActive, alarms.StatusNoted, alarms.StatusCleared:
		default:
			return apiutil.ErrInvalidAlarmStatus
		}
	}

	return nil
}

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

	return validatePageMetadata(req.pageMetadata)
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

	return validatePageMetadata(req.pageMetadata)
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

	return validatePageMetadata(req.pageMetadata)
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
	case alarms.StatusNoted, alarms.StatusCleared:
	default:
		return apiutil.ErrInvalidAlarmStatus
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

	return validatePageMetadata(req.pageMetadata)
}
