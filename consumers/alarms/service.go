package alarms

import (
	"context"
	"encoding/json"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/consumers"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
)

const (
	Viewer = "viewer"
)

type Service interface {
	ListAlarmsByGroup(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (AlarmsPage, error)
	ListAlarmsByThing(ctx context.Context, token, thingID string, pm apiutil.PageMetadata) (AlarmsPage, error)
	ListAlarmsByOrg(ctx context.Context, token, orgID string, pm apiutil.PageMetadata) (AlarmsPage, error)
	ViewAlarm(ctx context.Context, token, id string) (Alarm, error)
	RemoveAlarms(ctx context.Context, token string, id ...string) error
	consumers.Consumer
}

type alarmService struct {
	auth       protomfx.AuthServiceClient
	things     protomfx.ThingsServiceClient
	alarms     AlarmRepository
	idProvider uuid.IDProvider
}

var _ Service = (*alarmService)(nil)

func New(auth protomfx.AuthServiceClient, things protomfx.ThingsServiceClient, alarms AlarmRepository, idp uuid.IDProvider) Service {
	return &alarmService{
		auth:       auth,
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
	if err := as.isAdmin(ctx, token); err == nil {
		res, err := as.things.GetGroupIDsByOrg(ctx, &protomfx.OrgID{Value: orgID})
		if err != nil {
			return AlarmsPage{}, err
		}

		return as.alarms.RetrieveByGroups(ctx, res.GetIds(), pm)
	}

	if err := as.canAccessOrg(ctx, token, orgID, auth.OrgSub, Viewer); err != nil {
		return AlarmsPage{}, err
	}

	user, err := as.auth.Identify(ctx, &protomfx.Token{Value: token})
	if err != nil {
		return AlarmsPage{}, errors.Wrap(errors.ErrAuthentication, err)
	}

	res, err := as.things.GetGroupIDsByOrgMembership(ctx, &protomfx.OrgMembershipReq{
		OrgId:  orgID,
		UserId: user.GetId(),
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

func (as *alarmService) createAlarm(ctx context.Context, alarm *Alarm) error {
	grID, err := as.things.GetGroupIDByThingID(ctx, &protomfx.ThingID{Value: alarm.ThingID})
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

func (as *alarmService) Consume(message interface{}) error {
	ctx := context.Background()

	if msg, ok := message.(protomfx.Message); ok {
		var payload map[string]interface{}
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			return err
		}

		alarm := Alarm{
			ThingID:  msg.Publisher,
			Subtopic: msg.Subtopic,
			Protocol: msg.Protocol,
			Payload:  payload,
			Created:  msg.Created,
		}

		if err := as.createAlarm(ctx, &alarm); err != nil {
			return err
		}
	}

	return nil
}

func (as *alarmService) isAdmin(ctx context.Context, token string) error {
	req := &protomfx.AuthorizeReq{
		Token:   token,
		Subject: auth.RootSub,
	}

	if _, err := as.auth.Authorize(ctx, req); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	return nil
}

func (as *alarmService) canAccessOrg(ctx context.Context, token, orgID, subject, action string) error {
	req := &protomfx.AuthorizeReq{
		Token:   token,
		Object:  orgID,
		Subject: subject,
		Action:  action,
	}

	if _, err := as.auth.Authorize(ctx, req); err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}

	return nil
}
