package alarms

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MainfluxLabs/mainflux/consumers"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
)

var AllowedOrders = map[string]string{
	"id":      "id",
	"created": "created",
}

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
// All methods that accept a token parameter use it to identify and authorize
// the user performing the operation.
type Service interface {
	// ListAlarmsByGroup retrieves data about a subset of alarms
	// related to a certain group, identified by the provided group ID.
	ListAlarmsByGroup(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (AlarmsPage, error)

	// ListAlarmsByThing retrieves data about a subset of alarms
	// related to a certain thing, identified by the provided thing ID.
	ListAlarmsByThing(ctx context.Context, token, thingID string, pm apiutil.PageMetadata) (AlarmsPage, error)

	// ListAlarmsByOrg retrieves data about a subset of alarms
	// related to a certain organization, identified by the provided organization ID.
	ListAlarmsByOrg(ctx context.Context, token, orgID string, pm apiutil.PageMetadata) (AlarmsPage, error)

	// ViewAlarm retrieves data about the alarm identified by the provided ID.
	ViewAlarm(ctx context.Context, token, id string) (Alarm, error)

	// RemoveAlarms removes alarms identified with the provided IDs.
	RemoveAlarms(ctx context.Context, token string, id ...string) error

	// RemoveAlarmsByThing removes alarms related to the specified thing,
	// identified by the provided thing ID.
	RemoveAlarmsByThing(ctx context.Context, thingID string) error

	// RemoveAlarmsByGroup removes alarms related to the specified group,
	// identified by the provided group ID.
	RemoveAlarmsByGroup(ctx context.Context, groupID string) error

	// ExportAlarmsByThing retrieves a subset of alarms related to the specified thing
	// identified by the provided thing ID, intended for exporting.
	ExportAlarmsByThing(ctx context.Context, token, thingID string, pm apiutil.PageMetadata) (AlarmsPage, error)

	consumers.Consumer
}

type alarmService struct {
	things     protomfx.ThingsServiceClient
	alarms     AlarmRepository
	idProvider uuid.IDProvider
}

var _ Service = (*alarmService)(nil)

func New(things protomfx.ThingsServiceClient, alarms AlarmRepository, idp uuid.IDProvider) Service {
	return &alarmService{
		things:     things,
		alarms:     alarms,
		idProvider: idp,
	}
}

func (as *alarmService) ListAlarmsByGroup(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (AlarmsPage, error) {
	_, err := as.things.CanUserAccessGroup(ctx, &protomfx.UserAccessReq{Token: token, Id: groupID, Action: things.Viewer})
	if err != nil {
		return AlarmsPage{}, err
	}

	alarms, err := as.alarms.RetrieveByGroup(ctx, groupID, pm)
	if err != nil {
		return AlarmsPage{}, err
	}

	return alarms, nil
}

func (as *alarmService) ListAlarmsByThing(ctx context.Context, token, thingID string, pm apiutil.PageMetadata) (AlarmsPage, error) {
	_, err := as.things.CanUserAccessThing(ctx, &protomfx.UserAccessReq{Token: token, Id: thingID, Action: things.Viewer})
	if err != nil {
		return AlarmsPage{}, err
	}

	alarms, err := as.alarms.RetrieveByThing(ctx, thingID, pm)
	if err != nil {
		return AlarmsPage{}, err
	}

	return alarms, nil
}

func (as *alarmService) ListAlarmsByOrg(ctx context.Context, token string, orgID string, pm apiutil.PageMetadata) (AlarmsPage, error) {
	res, err := as.things.GetGroupIDsByOrg(ctx, &protomfx.OrgAccessReq{
		OrgId: orgID,
		Token: token,
	})
	if err != nil {
		return AlarmsPage{}, err
	}

	return as.alarms.RetrieveByGroups(ctx, res.GetIds(), pm)
}

func (as *alarmService) ViewAlarm(ctx context.Context, token, id string) (Alarm, error) {
	alarm, err := as.alarms.RetrieveByID(ctx, id)
	if err != nil {
		return Alarm{}, err
	}

	if _, err := as.things.CanUserAccessThing(ctx, &protomfx.UserAccessReq{Token: token, Id: alarm.ThingID, Action: things.Viewer}); err != nil {
		return Alarm{}, err
	}

	return alarm, nil
}

func (as *alarmService) RemoveAlarms(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		alarm, err := as.alarms.RetrieveByID(ctx, id)
		if err != nil {
			return err
		}
		if _, err := as.things.CanUserAccessThing(ctx, &protomfx.UserAccessReq{Token: token, Id: alarm.ThingID, Action: things.Editor}); err != nil {
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

func (as *alarmService) ExportAlarmsByThing(ctx context.Context, token, thingID string, pm apiutil.PageMetadata) (AlarmsPage, error) {
	_, err := as.things.CanUserAccessThing(ctx, &protomfx.UserAccessReq{Token: token, Id: thingID, Action: things.Viewer})
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
	grID, err := as.things.GetGroupIDByThing(ctx, &protomfx.ThingID{Value: alarm.ThingID})
	if err != nil {
		return err
	}
	alarm.GroupID = grID.GetValue()

	id, err := as.idProvider.ID()
	if err != nil {
		return err
	}
	alarm.ID = id

	return as.alarms.Save(ctx, *alarm)

}

func (as *alarmService) Consume(message any) error {
	ctx := context.Background()

	msg, ok := message.(protomfx.Message)
	if !ok {
		return errors.ErrMessage
	}

	subject := strings.Split(msg.Subject, ".")
	if len(subject) < 2 {
		return fmt.Errorf("invalid subject: %s", msg.Subject)
	}
	ruleID := subject[1]

	var payload map[string]any
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}

	alarm := Alarm{
		ThingID:  msg.Publisher,
		RuleID:   ruleID,
		Subtopic: msg.Subtopic,
		Protocol: msg.Protocol,
		Payload:  payload,
		Created:  msg.Created,
	}

	if err := as.createAlarm(ctx, &alarm); err != nil {
		return err
	}

	return nil
}
