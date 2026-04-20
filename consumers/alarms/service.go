package alarms

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MainfluxLabs/mainflux/consumers"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
)

var AllowedOrders = map[string]string{
	"id":      "id",
	"created": "created",
}

// PageMetadata contains page metadata that helps navigation.
type PageMetadata struct {
	Total   uint64         `json:"total,omitempty"`
	Offset  uint64         `json:"offset,omitempty"`
	Limit   uint64         `json:"limit,omitempty"`
	Order   string         `json:"order,omitempty"`
	Dir     string         `json:"dir,omitempty"`
	Payload map[string]any `json:"payload,omitempty"`
}

// Validate validates the page metadata.
func (pm PageMetadata) Validate(maxLimitSize int) error {
	common := apiutil.PageMetadata{Offset: pm.Offset, Limit: pm.Limit, Order: pm.Order, Dir: pm.Dir}
	return common.Validate(maxLimitSize, AllowedOrders)
}

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
// All methods that accept a token parameter use it to identify and authorize
// the user performing the operation.
type Service interface {
	// ListAlarmsByGroup retrieves data about a subset of alarms
	// related to a certain group, identified by the provided group ID.
	ListAlarmsByGroup(ctx context.Context, token, groupID string, pm PageMetadata) (AlarmsPage, error)

	// ListAlarmsByThing retrieves data about a subset of alarms
	// related to a certain thing, identified by the provided thing ID.
	ListAlarmsByThing(ctx context.Context, token, thingID string, pm PageMetadata) (AlarmsPage, error)

	// ListAlarmsByOrg retrieves data about a subset of alarms
	// related to a certain organization, identified by the provided organization ID.
	ListAlarmsByOrg(ctx context.Context, token, orgID string, pm PageMetadata) (AlarmsPage, error)

	// ViewAlarm retrieves data about the alarm identified by the provided ID.
	ViewAlarm(ctx context.Context, token, id string) (Alarm, error)

	// RemoveAlarms removes alarms identified with the provided IDs.
	RemoveAlarms(ctx context.Context, token string, id ...string) error

	// UpdateAlarmStatus updates the status of the alarm identified by the provided ID.
	UpdateAlarmStatus(ctx context.Context, token, id, status string) error

	// RemoveAlarmsByThing removes alarms related to the specified thing,
	// identified by the provided thing ID.
	RemoveAlarmsByThing(ctx context.Context, thingID string) error

	// RemoveAlarmsByGroup removes alarms related to the specified group,
	// identified by the provided group ID.
	RemoveAlarmsByGroup(ctx context.Context, groupID string) error

	// ExportAlarmsByThing retrieves a subset of alarms related to the specified thing
	// identified by the provided thing ID, intended for exporting.
	ExportAlarmsByThing(ctx context.Context, token, thingID string, pm PageMetadata) (AlarmsPage, error)

	consumers.AlarmConsumer
}

type alarmService struct {
	things     domain.ThingsClient
	alarms     AlarmRepository
	idProvider uuid.IDProvider
}

var _ Service = (*alarmService)(nil)

func New(things domain.ThingsClient, alarms AlarmRepository, idp uuid.IDProvider) Service {
	return &alarmService{
		things:     things,
		alarms:     alarms,
		idProvider: idp,
	}
}

func (as *alarmService) ListAlarmsByGroup(ctx context.Context, token, groupID string, pm PageMetadata) (AlarmsPage, error) {
	err := as.things.CanUserAccessGroup(ctx, domain.UserAccessReq{Token: token, ID: groupID, Action: domain.GroupViewer})
	if err != nil {
		return AlarmsPage{}, err
	}

	alarms, err := as.alarms.RetrieveByGroup(ctx, groupID, pm)
	if err != nil {
		return AlarmsPage{}, err
	}

	return alarms, nil
}

func (as *alarmService) ListAlarmsByThing(ctx context.Context, token, thingID string, pm PageMetadata) (AlarmsPage, error) {
	err := as.things.CanUserAccessThing(ctx, domain.UserAccessReq{Token: token, ID: thingID, Action: domain.GroupViewer})
	if err != nil {
		return AlarmsPage{}, err
	}

	alarms, err := as.alarms.RetrieveByThing(ctx, thingID, pm)
	if err != nil {
		return AlarmsPage{}, err
	}

	return alarms, nil
}

func (as *alarmService) ListAlarmsByOrg(ctx context.Context, token string, orgID string, pm PageMetadata) (AlarmsPage, error) {
	groupIDs, err := as.things.GetGroupIDsByOrg(ctx, domain.OrgAccessReq{OrgID: orgID, Token: token})
	if err != nil {
		return AlarmsPage{}, err
	}

	return as.alarms.RetrieveByGroups(ctx, groupIDs, pm)
}

func (as *alarmService) ViewAlarm(ctx context.Context, token, id string) (Alarm, error) {
	alarm, err := as.alarms.RetrieveByID(ctx, id)
	if err != nil {
		return Alarm{}, err
	}

	if err := as.things.CanUserAccessThing(ctx, domain.UserAccessReq{Token: token, ID: alarm.ThingID, Action: domain.GroupViewer}); err != nil {
		return Alarm{}, err
	}

	return alarm, nil
}

func (as *alarmService) UpdateAlarmStatus(ctx context.Context, token, id, status string) error {
	alarm, err := as.alarms.RetrieveByID(ctx, id)
	if err != nil {
		return err
	}

	if err := as.things.CanUserAccessThing(ctx, domain.UserAccessReq{Token: token, ID: alarm.ThingID, Action: domain.GroupEditor}); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	return as.alarms.UpdateStatus(ctx, id, status)
}

func (as *alarmService) RemoveAlarms(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		alarm, err := as.alarms.RetrieveByID(ctx, id)
		if err != nil {
			return err
		}
		if err := as.things.CanUserAccessThing(ctx, domain.UserAccessReq{Token: token, ID: alarm.ThingID, Action: domain.GroupEditor}); err != nil {
			return errors.Wrap(errors.ErrAuthorization, err)
		}
	}

	return as.alarms.Remove(ctx, ids...)
}

func (as *alarmService) RemoveAlarmsByThing(ctx context.Context, thingID string) error {
	return as.alarms.RemoveByThing(ctx, thingID)
}

func (as *alarmService) RemoveAlarmsByGroup(ctx context.Context, groupID string) error {
	return as.alarms.RemoveByGroup(ctx, groupID)
}

func (as *alarmService) ExportAlarmsByThing(ctx context.Context, token, thingID string, pm PageMetadata) (AlarmsPage, error) {
	err := as.things.CanUserAccessThing(ctx, domain.UserAccessReq{Token: token, ID: thingID, Action: domain.GroupViewer})
	if err != nil {
		return AlarmsPage{}, err
	}

	alarms, err := as.alarms.ExportByThing(ctx, thingID, pm)
	if err != nil {
		return AlarmsPage{}, err
	}

	return alarms, nil
}

func (as *alarmService) createAlarm(ctx context.Context, alarm *Alarm) error {
	grID, err := as.things.GetGroupIDByThing(ctx, alarm.ThingID)
	if err != nil {
		return err
	}
	alarm.GroupID = grID

	id, err := as.idProvider.ID()
	if err != nil {
		return err
	}
	alarm.ID = id

	return as.alarms.Save(ctx, *alarm)

}

func (as *alarmService) ConsumeAlarm(subject string, alarm protomfx.Alarm) error {
	ctx := context.Background()

	subParts := strings.Split(subject, ".")
	if len(subParts) < 3 {
		return errors.ErrInvalidSubject
	}

	originType := subParts[1]
	originID := subParts[2]

	var ruleInfo *RuleInfo
	if len(alarm.RuleInfo) > 0 {
		var ri RuleInfo
		if err := json.Unmarshal(alarm.RuleInfo, &ri); err != nil {
			return err
		}
		ruleInfo = &ri
	}

	a := Alarm{
		ThingID:  alarm.ThingId,
		Subtopic: alarm.Subtopic,
		Protocol: alarm.Protocol,
		Rule:     ruleInfo,
		Level:    alarm.Level,
		Status:   AlarmStatusActive,
	}

	switch originType {
	case domain.AlarmOriginRule:
		a.RuleID = originID
	case domain.AlarmOriginScript:
		a.ScriptID = originID
	default:
		return fmt.Errorf("invalid subject origin type: %s", originType)
	}

	return as.createAlarm(ctx, &a)
}
