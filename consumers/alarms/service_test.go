package alarms_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/consumers/alarms"
	"github.com/MainfluxLabs/mainflux/consumers/alarms/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	authmock "github.com/MainfluxLabs/mainflux/pkg/mocks"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	token      = "admin@example.com"
	wrongValue = "wrong-value"
	thingID    = "5384fb1c-d0ae-4cbe-be52-c54223150fe0"
	groupID    = "574106f7-030e-4881-8ab0-151195c29f94"
	orgID      = "7e3d5e48-b0b4-4d7b-9d6a-c81f40e30e2c"
	ruleID     = "5384fb1c-d0ae-4cbe-be52-c54223150fe1"
	subtopic   = "sensors"
	protocol   = "mqtt"
)

var payload = map[string]any{"temperature": float64(30), "humidity": float64(60)}

func newService() alarms.Service {
	ths := authmock.NewThingsServiceClient(
		nil,
		map[string]things.Thing{
			token:   {ID: thingID, GroupID: groupID},
			thingID: {ID: thingID, GroupID: groupID},
		},
		map[string]things.Group{
			token: {ID: groupID, OrgID: orgID},
		},
	)
	alarmRepo := mocks.NewAlarmRepository()
	idProvider := uuid.NewMock()

	return alarms.New(ths, alarmRepo, idProvider)
}

func saveAlarms(t *testing.T, svc alarms.Service, n int) []alarms.Alarm {
	t.Helper()

	pyd, err := json.Marshal(payload)
	require.Nil(t, err)

	var saved []alarms.Alarm
	for i := 0; i < n; i++ {
		msg := protomfx.Message{
			Publisher: thingID,
			Subject:   fmt.Sprintf("alarms.%s", ruleID),
			Subtopic:  subtopic,
			Protocol:  protocol,
			Payload:   pyd,
			Created:   int64(1000000 + i),
		}
		err := svc.Consume(msg)
		require.Nil(t, err, fmt.Sprintf("unexpected error saving alarm %d: %s", i+1, err))

		// build expected alarm for later assertions
		saved = append(saved, alarms.Alarm{
			ThingID:  thingID,
			GroupID:  groupID,
			RuleID:   ruleID,
			Subtopic: subtopic,
			Protocol: protocol,
			Payload:  payload,
			Created:  int64(1000000 + i),
		})
	}

	return saved
}

func TestConsume(t *testing.T) {
	svc := newService()

	pyd, err := json.Marshal(payload)
	require.Nil(t, err)

	validMsg := protomfx.Message{
		Publisher: thingID,
		Subject:   fmt.Sprintf("alarms.%s", ruleID),
		Subtopic:  subtopic,
		Protocol:  protocol,
		Payload:   pyd,
		Created:   1717430400,
	}

	invalidPayloadMsg := validMsg
	invalidPayloadMsg.Payload = []byte("invalid")

	invalidSubjectMsg := validMsg
	invalidSubjectMsg.Subject = "invalid"

	unknownThingMsg := validMsg
	unknownThingMsg.Publisher = wrongValue

	cases := []struct {
		desc string
		msg  any
		err  error
	}{
		{
			desc: "consume valid message",
			msg:  validMsg,
			err:  nil,
		},
		{
			desc: "consume message with invalid payload",
			msg:  invalidPayloadMsg,
			err:  errors.New("invalid character"),
		},
		{
			desc: "consume message with invalid subject",
			msg:  invalidSubjectMsg,
			err:  errors.New("invalid subject"),
		},
		{
			desc: "consume message with unknown thing",
			msg:  unknownThingMsg,
			err:  dbutil.ErrNotFound,
		},
		{
			desc: "consume non-message type",
			msg:  "not-a-message",
			err:  errors.ErrMessage,
		},
	}

	for _, tc := range cases {
		err := svc.Consume(tc.msg)
		if tc.err == nil {
			assert.Nil(t, err, fmt.Sprintf("%s: expected no error, got %s", tc.desc, err))
		} else {
			assert.NotNil(t, err, fmt.Sprintf("%s: expected error, got nil", tc.desc))
		}
	}
}

func TestListAlarmsByGroup(t *testing.T) {
	svc := newService()
	n := 10
	saveAlarms(t, svc, n)

	cases := []struct {
		desc         string
		token        string
		groupID      string
		pageMetadata apiutil.PageMetadata
		size         uint64
		err          error
	}{
		{
			desc:    "list alarms by group",
			token:   token,
			groupID: groupID,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  uint64(n),
			},
			size: uint64(n),
			err:  nil,
		},
		{
			desc:    "list alarms by group with no limit",
			token:   token,
			groupID: groupID,
			pageMetadata: apiutil.PageMetadata{
				Limit: 0,
			},
			size: uint64(n),
			err:  nil,
		},
		{
			desc:    "list last alarm by group",
			token:   token,
			groupID: groupID,
			pageMetadata: apiutil.PageMetadata{
				Offset: uint64(n) - 1,
				Limit:  uint64(n),
			},
			size: 1,
			err:  nil,
		},
		{
			desc:    "list empty set of alarms by group",
			token:   token,
			groupID: groupID,
			pageMetadata: apiutil.PageMetadata{
				Offset: uint64(n) + 1,
				Limit:  uint64(n),
			},
			size: 0,
			err:  nil,
		},
		{
			desc:    "list alarms by group with invalid auth token",
			token:   wrongValue,
			groupID: groupID,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  uint64(n),
			},
			size: 0,
			err:  errors.ErrAuthentication,
		},
		{
			desc:    "list alarms by group with invalid group ID",
			token:   token,
			groupID: wrongValue,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  uint64(n),
			},
			size: 0,
			err:  errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		page, err := svc.ListAlarmsByGroup(context.Background(), tc.token, tc.groupID, tc.pageMetadata)
		size := uint64(len(page.Alarms))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d", tc.desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}

func TestListAlarmsByThing(t *testing.T) {
	svc := newService()
	n := 10
	saveAlarms(t, svc, n)

	cases := []struct {
		desc         string
		token        string
		thingID      string
		pageMetadata apiutil.PageMetadata
		size         uint64
		err          error
	}{
		{
			desc:    "list alarms by thing",
			token:   token,
			thingID: thingID,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  uint64(n),
			},
			size: uint64(n),
			err:  nil,
		},
		{
			desc:    "list alarms by thing with no limit",
			token:   token,
			thingID: thingID,
			pageMetadata: apiutil.PageMetadata{
				Limit: 0,
			},
			size: uint64(n),
			err:  nil,
		},
		{
			desc:    "list last alarm by thing",
			token:   token,
			thingID: thingID,
			pageMetadata: apiutil.PageMetadata{
				Offset: uint64(n) - 1,
				Limit:  uint64(n),
			},
			size: 1,
			err:  nil,
		},
		{
			desc:    "list empty set of alarms by thing",
			token:   token,
			thingID: thingID,
			pageMetadata: apiutil.PageMetadata{
				Offset: uint64(n) + 1,
				Limit:  uint64(n),
			},
			size: 0,
			err:  nil,
		},
		{
			desc:    "list alarms by thing with invalid auth token",
			token:   wrongValue,
			thingID: thingID,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  uint64(n),
			},
			size: 0,
			err:  errors.ErrAuthentication,
		},
		{
			desc:    "list alarms by thing with invalid thing ID",
			token:   token,
			thingID: wrongValue,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  uint64(n),
			},
			size: 0,
			err:  errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		page, err := svc.ListAlarmsByThing(context.Background(), tc.token, tc.thingID, tc.pageMetadata)
		size := uint64(len(page.Alarms))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d", tc.desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}

func TestListAlarmsByOrg(t *testing.T) {
	svc := newService()
	n := 5
	saveAlarms(t, svc, n)

	cases := []struct {
		desc         string
		token        string
		orgID        string
		pageMetadata apiutil.PageMetadata
		size         uint64
		err          error
	}{
		{
			desc:  "list alarms by org",
			token: token,
			orgID: orgID,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  uint64(n),
			},
			size: uint64(n),
			err:  nil,
		},
		{
			desc:  "list alarms by org with no limit",
			token: token,
			orgID: orgID,
			pageMetadata: apiutil.PageMetadata{
				Limit: 0,
			},
			size: uint64(n),
			err:  nil,
		},
		{
			desc:  "list alarms by org with wrong org ID returns empty",
			token: token,
			orgID: wrongValue,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  uint64(n),
			},
			size: 0,
			err:  nil,
		},
	}

	for _, tc := range cases {
		page, err := svc.ListAlarmsByOrg(context.Background(), tc.token, tc.orgID, tc.pageMetadata)
		size := uint64(len(page.Alarms))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d", tc.desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}

func TestViewAlarm(t *testing.T) {
	svc := newService()
	saveAlarms(t, svc, 1)

	page, err := svc.ListAlarmsByThing(context.Background(), token, thingID, apiutil.PageMetadata{Limit: 10})
	require.Nil(t, err)
	require.Equal(t, 1, len(page.Alarms))
	alarmID := page.Alarms[0].ID

	cases := []struct {
		desc  string
		token string
		id    string
		err   error
	}{
		{
			desc:  "view existing alarm",
			token: token,
			id:    alarmID,
			err:   nil,
		},
		{
			desc:  "view alarm with wrong credentials",
			token: wrongValue,
			id:    alarmID,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "view non-existing alarm",
			token: token,
			id:    wrongValue,
			err:   dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		_, err := svc.ViewAlarm(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}

func TestRemoveAlarms(t *testing.T) {
	svc := newService()
	saveAlarms(t, svc, 1)

	page, err := svc.ListAlarmsByThing(context.Background(), token, thingID, apiutil.PageMetadata{Limit: 10})
	require.Nil(t, err)
	require.Equal(t, 1, len(page.Alarms))
	alarmID := page.Alarms[0].ID

	cases := []struct {
		desc  string
		token string
		ids   []string
		err   error
	}{
		{
			desc:  "remove alarm with wrong credentials",
			token: wrongValue,
			ids:   []string{alarmID},
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "remove existing alarm",
			token: token,
			ids:   []string{alarmID},
			err:   nil,
		},
		{
			desc:  "remove non-existing alarm",
			token: token,
			ids:   []string{wrongValue},
			err:   dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveAlarms(context.Background(), tc.token, tc.ids...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}

func TestRemoveAlarmsByThing(t *testing.T) {
	svc := newService()
	n := 3
	saveAlarms(t, svc, n)

	page, err := svc.ListAlarmsByThing(context.Background(), token, thingID, apiutil.PageMetadata{Limit: 20})
	require.Nil(t, err)
	require.Equal(t, n, len(page.Alarms))

	cases := []struct {
		desc    string
		thingID string
		err     error
	}{
		{
			desc:    "remove alarms by thing",
			thingID: thingID,
			err:     nil,
		},
		{
			desc:    "remove alarms by non-existing thing",
			thingID: wrongValue,
			err:     nil,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveAlarmsByThing(context.Background(), tc.thingID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}

	page, err = svc.ListAlarmsByThing(context.Background(), token, thingID, apiutil.PageMetadata{})
	require.Nil(t, err)
	assert.Equal(t, 0, len(page.Alarms), "expected no alarms after removal by thing")
}

func TestRemoveAlarmsByGroup(t *testing.T) {
	svc := newService()
	n := 3
	saveAlarms(t, svc, n)

	page, err := svc.ListAlarmsByGroup(context.Background(), token, groupID, apiutil.PageMetadata{Limit: 20})
	require.Nil(t, err)
	require.Equal(t, n, len(page.Alarms))

	cases := []struct {
		desc    string
		groupID string
		err     error
	}{
		{
			desc:    "remove alarms by group",
			groupID: groupID,
			err:     nil,
		},
		{
			desc:    "remove alarms by non-existing group",
			groupID: wrongValue,
			err:     nil,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveAlarmsByGroup(context.Background(), tc.groupID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}

	page, err = svc.ListAlarmsByGroup(context.Background(), token, groupID, apiutil.PageMetadata{})
	require.Nil(t, err)
	assert.Equal(t, 0, len(page.Alarms), "expected no alarms after removal by group")
}

func TestExportAlarmsByThing(t *testing.T) {
	svc := newService()
	n := 10
	saveAlarms(t, svc, n)

	cases := []struct {
		desc         string
		token        string
		thingID      string
		pageMetadata apiutil.PageMetadata
		size         uint64
		err          error
	}{
		{
			desc:    "export alarms by thing",
			token:   token,
			thingID: thingID,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  uint64(n),
			},
			size: uint64(n),
			err:  nil,
		},
		{
			desc:    "export alarms by thing with no limit",
			token:   token,
			thingID: thingID,
			pageMetadata: apiutil.PageMetadata{
				Limit: 0,
			},
			size: uint64(n),
			err:  nil,
		},
		{
			desc:    "export alarms by thing with invalid auth token",
			token:   wrongValue,
			thingID: thingID,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  uint64(n),
			},
			size: 0,
			err:  errors.ErrAuthentication,
		},
		{
			desc:    "export alarms by thing with invalid thing ID",
			token:   token,
			thingID: wrongValue,
			pageMetadata: apiutil.PageMetadata{
				Offset: 0,
				Limit:  uint64(n),
			},
			size: 0,
			err:  errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		page, err := svc.ExportAlarmsByThing(context.Background(), tc.token, tc.thingID, tc.pageMetadata)
		size := uint64(len(page.Alarms))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d", tc.desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}
