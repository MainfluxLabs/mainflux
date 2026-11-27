// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package events_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/modbus"
	mbevents "github.com/MainfluxLabs/mainflux/modbus/events"
	mbmocks "github.com/MainfluxLabs/mainflux/modbus/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/authn"
	"github.com/MainfluxLabs/mainflux/pkg/cron"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/events"
	pkgmocks "github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/users"
	r "github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	thingID         = "5384fb1c-d0ae-4cbe-be52-c54223150fe0"
	groupID         = "c5a1e8f2-6b3d-4a9e-8f7c-2d1b0e9a8c7f"
	orgID           = "f8d2c1b4-7a6e-4d3c-9b8a-5e4f3d2c1b0a"
	orphanThingID   = "a6e1c2f4-3b5d-4f7e-9c8a-1d2e3f4a5b6c"
	orphanGroupID   = "b7f2d3e5-4c6e-4a8f-8d9b-2e3f4a5b6c7d"
	clientName      = "client"
	ipAddress       = "192.168.1.1"
	port            = "502"
	updatedName     = "updated-client"
	invalidToken    = "invalid"
	unknownClient   = "unknown-client"
	orphanUserToken = "orphan@example.com"
)

var (
	user = users.User{
		ID:    "574106f7-030e-4881-8ab0-151195c29f94",
		Email: "user@example.com",
	}
	// orphanUser edits Modbus clients in a group whose org can't be looked up.
	orphanUser = users.User{
		ID:    "e0d1c2b3-a495-4867-8f9a-0b1c2d3e4f5a",
		Email: orphanUserToken,
	}
	metadata   = map[string]any{"key": "value"}
	scheduler  = cron.Scheduler{Frequency: cron.MinutelyFreq, Minute: 5, TimeZone: "UTC"}
	dataFields = []modbus.DataField{{Name: "temperature", Type: modbus.Float32Type, ByteOrder: modbus.ByteOrderABCD}}
)

type testEnv struct {
	svc  modbus.Service
	auth domain.AuthClient
}

func newEnv(t *testing.T) testEnv {
	t.Helper()
	ac := pkgmocks.NewAuthService("", []users.User{user, orphanUser}, nil)

	// The things mock authorizes by token and looks things and groups up by ID.
	ths := map[string]domain.Thing{
		user.Email:      {ID: thingID, GroupID: groupID},
		thingID:         {ID: thingID, GroupID: groupID},
		orphanUserToken: {ID: orphanThingID, GroupID: orphanGroupID},
		orphanThingID:   {ID: orphanThingID, GroupID: orphanGroupID},
	}
	groups := map[string]domain.Group{
		user.Email:      {ID: groupID},
		groupID:         {ID: groupID, OrgID: orgID},
		orphanUserToken: {ID: orphanGroupID},
	}
	tc := pkgmocks.NewThingsServiceClient(nil, ths, groups)

	pub, err := events.NewPublisher(events.PublisherConfig{
		URL:                  redisURL,
		Stream:               events.ModbusStream,
		BufferSize:           events.DefBufferSize,
		DrainInitialInterval: 10 * time.Millisecond,
		DrainMaxBackoff:      100 * time.Millisecond,
		ShutdownDrainTimeout: 2 * time.Second,
	}, logger.NewMock())
	require.NoError(t, err)
	t.Cleanup(func() { _ = pub.Close() })

	svc := modbus.New(tc, pkgmocks.NewPublisher(), mbmocks.NewClientRepository(), uuid.NewMock(), logger.NewMock())

	return testEnv{
		svc:  mbevents.NewEventStoreMiddleware(svc, tc, pub),
		auth: ac,
	}
}

// identityCtx returns a context carrying the token owner's identity, as the HTTP API sets it.
func (env testEnv) identityCtx(token string) context.Context {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	ctx := authn.HTTPTokenToContext(context.Background(), req)

	var identityCtx context.Context
	capture := func(ctx context.Context, _ any) (any, error) {
		identityCtx = ctx
		return nil, nil
	}
	_, _ = authn.IdentityMiddleware(env.auth, logger.NewMock())(capture)(ctx, nil)

	return identityCtx
}

func (env testEnv) createClient(t *testing.T, token, thingID, name string) modbus.Client {
	t.Helper()
	cl := modbus.Client{
		Name:         name,
		IPAddress:    ipAddress,
		Port:         port,
		SlaveID:      1,
		FunctionCode: modbus.ReadHoldingRegistersFunc,
		Scheduler:    scheduler,
		DataFields:   dataFields,
		Metadata:     metadata,
	}
	cls, err := env.svc.CreateClients(context.Background(), token, thingID, cl)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	return cls[0]
}

func nextEvent(t *testing.T, desc, lastID string) (map[string]any, string) {
	t.Helper()
	streams := redisClient.XRead(context.Background(), &r.XReadArgs{
		Streams: []string{events.ModbusStream, lastID},
		Count:   1,
		Block:   time.Second,
	}).Val()

	if len(streams) == 0 || len(streams[0].Messages) == 0 {
		return nil, lastID
	}

	msg := streams[0].Messages[0]
	assert.NotEmpty(t, msg.Values["occurred_at"], desc)
	delete(msg.Values, "occurred_at")

	return msg.Values, msg.ID
}

// skipEvents reads past the given number of events, such as those of a test's setup.
func skipEvents(t *testing.T, count int) string {
	t.Helper()
	lastID := "0"
	for i := 0; i < count; i++ {
		_, lastID = nextEvent(t, "setup", lastID)
	}

	return lastID
}

// clientEvent returns the expected fields of an event the actor performed on a Modbus client in the given group and org.
func clientEvent(operation string, actor users.User, groupID, orgID string, fields map[string]any) map[string]any {
	event := map[string]any{
		"operation":               operation,
		"jwt_identity_user_id":    actor.ID,
		"jwt_identity_user_email": actor.Email,
		"evt_org_id":              orgID,
		"evt_group_id":            groupID,
	}
	for k, v := range fields {
		event[k] = v
	}

	return event
}

func TestCreateClients(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()
	env := newEnv(t)

	cases := []struct {
		desc    string
		actor   users.User
		token   string
		thingID string
		groupID string
		orgID   string
		err     error
	}{
		{
			desc:    "create client with invalid token",
			token:   invalidToken,
			thingID: thingID,
			err:     errors.ErrAuthentication,
		},
		{
			desc:    "create client",
			actor:   user,
			token:   user.Email,
			thingID: thingID,
			groupID: groupID,
			orgID:   orgID,
			err:     nil,
		},
		{
			desc:    "create client in a group whose org can't be looked up",
			actor:   orphanUser,
			token:   orphanUserToken,
			thingID: orphanThingID,
			groupID: orphanGroupID,
			orgID:   "",
			err:     nil,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		cl := modbus.Client{
			Name:         clientName,
			IPAddress:    ipAddress,
			Port:         port,
			SlaveID:      1,
			FunctionCode: modbus.ReadHoldingRegistersFunc,
			Scheduler:    scheduler,
			DataFields:   dataFields,
			Metadata:     metadata,
		}
		created, err := env.svc.CreateClients(env.identityCtx(tc.token), tc.token, tc.thingID, cl)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var expected map[string]any
		if tc.err == nil {
			expected = clientEvent(events.ModbusClientCreate, tc.actor, tc.groupID, tc.orgID, map[string]any{
				"id":       created[0].ID,
				"thing_id": tc.thingID,
				"name":     clientName,
				"metadata": `{"key":"value"}`,
			})
		}

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, expected, event)
		assert.Equal(t, expected, event, msg)
	}
}

func TestCreateClient(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()
	env := newEnv(t)

	cl := modbus.Client{
		Name:         clientName,
		IPAddress:    ipAddress,
		Port:         port,
		SlaveID:      1,
		FunctionCode: modbus.ReadHoldingRegistersFunc,
		Scheduler:    scheduler,
		DataFields:   dataFields,
		Metadata:     metadata,
	}

	cases := []struct {
		desc      string
		token     string
		err       error
		published bool
	}{
		{
			desc:      "create client with invalid token",
			token:     invalidToken,
			err:       errors.ErrAuthentication,
			published: false,
		},
		{
			desc:      "create client",
			token:     user.Email,
			err:       nil,
			published: true,
		},
	}

	lastID := "0"
	for _, tc := range cases {
		created, err := env.svc.CreateClient(env.identityCtx(tc.token), tc.token, thingID, cl)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var expected map[string]any
		if tc.published {
			expected = clientEvent(events.ModbusClientCreate, user, groupID, orgID, map[string]any{
				"id":       created.ID,
				"thing_id": thingID,
				"name":     clientName,
				"metadata": `{"key":"value"}`,
			})
		}

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, expected, event)
		assert.Equal(t, expected, event, msg)
	}
}

func TestUpdateClient(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()
	env := newEnv(t)
	cl := env.createClient(t, user.Email, thingID, clientName)

	update := modbus.Client{
		ID:           cl.ID,
		Name:         updatedName,
		IPAddress:    ipAddress,
		Port:         port,
		SlaveID:      1,
		FunctionCode: modbus.ReadHoldingRegistersFunc,
		Scheduler:    scheduler,
		DataFields:   dataFields,
		Metadata:     metadata,
	}

	cases := []struct {
		desc      string
		token     string
		client    modbus.Client
		err       error
		published bool
	}{
		{
			desc:      "update client with invalid token",
			token:     invalidToken,
			client:    update,
			err:       errors.ErrAuthentication,
			published: false,
		},
		{
			desc:      "update unknown client",
			token:     user.Email,
			client:    modbus.Client{ID: unknownClient, Name: updatedName},
			err:       dbutil.ErrNotFound,
			published: false,
		},
		{
			desc:      "update client",
			token:     user.Email,
			client:    update,
			err:       nil,
			published: true,
		},
	}

	lastID := skipEvents(t, 1)
	for _, tc := range cases {
		err := env.svc.UpdateClient(env.identityCtx(tc.token), tc.token, tc.client)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		var expected map[string]any
		if tc.published {
			expected = clientEvent(events.ModbusClientUpdate, user, groupID, orgID, map[string]any{
				"id":       cl.ID,
				"name":     updatedName,
				"metadata": `{"key":"value"}`,
			})
		}

		var event map[string]any
		event, lastID = nextEvent(t, tc.desc, lastID)
		msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, expected, event)
		assert.Equal(t, expected, event, msg)
	}
}

func TestRemoveClients(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()
	env := newEnv(t)
	first := env.createClient(t, user.Email, thingID, clientName)
	second := env.createClient(t, user.Email, thingID, updatedName)

	cases := []struct {
		desc    string
		token   string
		ids     []string
		err     error
		removed []string
	}{
		{
			desc:    "remove clients with invalid token",
			token:   invalidToken,
			ids:     []string{first.ID},
			err:     errors.ErrAuthentication,
			removed: nil,
		},
		{
			desc:    "remove clients including an unknown one",
			token:   user.Email,
			ids:     []string{first.ID, unknownClient},
			err:     dbutil.ErrNotFound,
			removed: nil,
		},
		{
			desc:    "remove clients",
			token:   user.Email,
			ids:     []string{first.ID, second.ID},
			err:     nil,
			removed: []string{first.ID, second.ID},
		},
	}

	lastID := skipEvents(t, 2)
	for _, tc := range cases {
		_, err := env.svc.RemoveClients(env.identityCtx(tc.token), tc.token, tc.ids...)
		msg := fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err)
		assert.True(t, errors.Contains(err, tc.err), msg)

		for _, id := range tc.removed {
			expected := clientEvent(events.ModbusClientRemove, user, groupID, orgID, map[string]any{"id": id})
			var event map[string]any
			event, lastID = nextEvent(t, tc.desc, lastID)
			msg = fmt.Sprintf("%s: expected %v got %v\n", tc.desc, expected, event)
			assert.Equal(t, expected, event, msg)
		}

		event, _ := nextEvent(t, tc.desc, lastID)
		assert.Nil(t, event, fmt.Sprintf("%s: unexpected event %v\n", tc.desc, event))
	}
}
