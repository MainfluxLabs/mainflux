// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package modbus_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/modbus"
	mbmocks "github.com/MainfluxLabs/mainflux/modbus/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/cron"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	pkgmocks "github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	token      = "admin@example.com"
	wrongToken = "wrong-token"
	thingID    = "5384fb1c-d0ae-4cbe-be52-c54223150fe0"
	groupID    = "574106f7-030e-4881-8ab0-151195c29f94"
	wrongID    = "wrong-id"
)

var client = modbus.Client{
	Name:         "test-client",
	IPAddress:    "192.168.1.1",
	Port:         "502",
	SlaveID:      1,
	FunctionCode: modbus.ReadHoldingRegistersFunc,
	Scheduler: cron.Scheduler{
		Frequency: cron.MinutelyFreq,
		Minute:    5,
		TimeZone:  "UTC",
	},
	DataFields: []modbus.DataField{
		{
			Name:      "temperature",
			Type:      modbus.Float32Type,
			Address:   0,
			ByteOrder: modbus.ByteOrderABCD,
		},
	},
}

func newService() modbus.Service {
	thingsSvc := pkgmocks.NewThingsServiceClient(
		nil,
		map[string]things.Thing{
			token:   {ID: thingID, GroupID: groupID},
			thingID: {ID: thingID, GroupID: groupID},
		},
		map[string]things.Group{token: {ID: groupID}},
	)
	repo := mbmocks.NewClientRepository()
	pub := pkgmocks.NewPublisher()
	idp := uuid.NewMock()
	log := logger.NewMock()

	return modbus.New(thingsSvc, pub, repo, idp, log)
}

func TestCreateClients(t *testing.T) {
	svc := newService()

	cases := []struct {
		desc    string
		token   string
		thingID string
		clients []modbus.Client
		err     error
	}{
		{
			desc:    "create clients with valid token",
			token:   token,
			thingID: thingID,
			clients: []modbus.Client{client},
			err:     nil,
		},
		{
			desc:    "create clients with invalid token",
			token:   wrongToken,
			thingID: thingID,
			clients: []modbus.Client{client},
			err:     errors.ErrAuthorization,
		},
		{
			desc:    "create clients for wrong thing ID",
			token:   token,
			thingID: wrongID,
			clients: []modbus.Client{client},
			err:     errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		cls, err := svc.CreateClients(context.Background(), tc.token, tc.thingID, tc.clients...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
		if tc.err == nil {
			assert.Equal(t, len(tc.clients), len(cls), fmt.Sprintf("%s: expected %d clients got %d", tc.desc, len(tc.clients), len(cls)))
		}
	}
}

func TestListClientsByThing(t *testing.T) {
	svc := newService()

	cls, err := svc.CreateClients(context.Background(), token, thingID, client)
	require.Nil(t, err, fmt.Sprintf("unexpected error creating clients: %s", err))
	require.Equal(t, 1, len(cls))

	cases := []struct {
		desc    string
		token   string
		thingID string
		pm      apiutil.PageMetadata
		size    uint64
		err     error
	}{
		{
			desc:    "list clients by thing with valid token",
			token:   token,
			thingID: thingID,
			pm:      apiutil.PageMetadata{},
			size:    1,
			err:     nil,
		},
		{
			desc:    "list clients by thing with invalid token",
			token:   wrongToken,
			thingID: thingID,
			pm:      apiutil.PageMetadata{},
			size:    0,
			err:     errors.ErrAuthorization,
		},
		{
			desc:    "list clients by thing for wrong thing ID",
			token:   token,
			thingID: wrongID,
			pm:      apiutil.PageMetadata{},
			size:    0,
			err:     errors.ErrAuthorization,
		},
		{
			desc:    "list clients by thing with limit",
			token:   token,
			thingID: thingID,
			pm:      apiutil.PageMetadata{Limit: 1, Offset: 0},
			size:    1,
			err:     nil,
		},
		{
			desc:    "list clients by thing with offset beyond available",
			token:   token,
			thingID: thingID,
			pm:      apiutil.PageMetadata{Limit: 1, Offset: 1},
			size:    0,
			err:     nil,
		},
	}

	for _, tc := range cases {
		page, err := svc.ListClientsByThing(context.Background(), tc.token, tc.thingID, tc.pm)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
		if tc.err == nil {
			assert.Equal(t, tc.size, uint64(len(page.Clients)), fmt.Sprintf("%s: expected %d clients got %d", tc.desc, tc.size, len(page.Clients)))
		}
	}
}

func TestListClientsByGroup(t *testing.T) {
	svc := newService()

	_, err := svc.CreateClients(context.Background(), token, thingID, client)
	require.Nil(t, err, fmt.Sprintf("unexpected error creating clients: %s", err))

	cases := []struct {
		desc    string
		token   string
		groupID string
		pm      apiutil.PageMetadata
		size    uint64
		err     error
	}{
		{
			desc:    "list clients by group with valid token",
			token:   token,
			groupID: groupID,
			pm:      apiutil.PageMetadata{},
			size:    1,
			err:     nil,
		},
		{
			desc:    "list clients by group with invalid token",
			token:   wrongToken,
			groupID: groupID,
			pm:      apiutil.PageMetadata{},
			size:    0,
			err:     errors.ErrAuthorization,
		},
		{
			desc:    "list clients by group for wrong group ID",
			token:   token,
			groupID: wrongID,
			pm:      apiutil.PageMetadata{},
			size:    0,
			err:     errors.ErrAuthorization,
		},
		{
			desc:    "list clients by group with limit",
			token:   token,
			groupID: groupID,
			pm:      apiutil.PageMetadata{Limit: 1, Offset: 0},
			size:    1,
			err:     nil,
		},
		{
			desc:    "list clients by group with offset beyond available",
			token:   token,
			groupID: groupID,
			pm:      apiutil.PageMetadata{Limit: 1, Offset: 1},
			size:    0,
			err:     nil,
		},
	}

	for _, tc := range cases {
		page, err := svc.ListClientsByGroup(context.Background(), tc.token, tc.groupID, tc.pm)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
		if tc.err == nil {
			assert.Equal(t, tc.size, uint64(len(page.Clients)), fmt.Sprintf("%s: expected %d clients got %d", tc.desc, tc.size, len(page.Clients)))
		}
	}
}

func TestViewClient(t *testing.T) {
	svc := newService()

	cls, err := svc.CreateClients(context.Background(), token, thingID, client)
	require.Nil(t, err, fmt.Sprintf("unexpected error creating clients: %s", err))
	require.Equal(t, 1, len(cls))
	clID := cls[0].ID

	cases := []struct {
		desc  string
		token string
		id    string
		err   error
	}{
		{
			desc:  "view client with valid token",
			token: token,
			id:    clID,
			err:   nil,
		},
		{
			desc:  "view client with invalid ID",
			token: token,
			id:    wrongID,
			err:   dbutil.ErrNotFound,
		},
		{
			desc:  "view client with invalid token",
			token: wrongToken,
			id:    clID,
			err:   errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		c, err := svc.ViewClient(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
		if tc.err == nil {
			assert.Equal(t, tc.id, c.ID, fmt.Sprintf("%s: expected ID %s got %s", tc.desc, tc.id, c.ID))
		}
	}
}

func TestUpdateClient(t *testing.T) {
	svc := newService()

	cls, err := svc.CreateClients(context.Background(), token, thingID, client)
	require.Nil(t, err, fmt.Sprintf("unexpected error creating clients: %s", err))
	require.Equal(t, 1, len(cls))
	clID := cls[0].ID

	updated := modbus.Client{
		ID:           clID,
		Name:         "updated-client",
		IPAddress:    "192.168.1.2",
		Port:         "503",
		SlaveID:      2,
		FunctionCode: modbus.ReadInputRegistersFunc,
		Scheduler: cron.Scheduler{
			Frequency: cron.HourlyFreq,
			Hour:      1,
			TimeZone:  "UTC",
		},
		DataFields: []modbus.DataField{
			{
				Name:      "pressure",
				Type:      modbus.Int16Type,
				Address:   1,
				ByteOrder: modbus.ByteOrderABCD,
			},
		},
	}

	cases := []struct {
		desc   string
		token  string
		client modbus.Client
		err    error
	}{
		{
			desc:   "update client with valid token",
			token:  token,
			client: updated,
			err:    nil,
		},
		{
			desc:   "update client with invalid ID",
			token:  token,
			client: modbus.Client{ID: wrongID, Name: "x", IPAddress: "1.1.1.1", Port: "502", FunctionCode: modbus.ReadHoldingRegistersFunc},
			err:    dbutil.ErrNotFound,
		},
		{
			desc:   "update client with invalid token",
			token:  wrongToken,
			client: updated,
			err:    errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		err := svc.UpdateClient(context.Background(), tc.token, tc.client)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}

func TestRemoveClients(t *testing.T) {
	svc := newService()

	cls, err := svc.CreateClients(context.Background(), token, thingID, client)
	require.Nil(t, err, fmt.Sprintf("unexpected error creating clients: %s", err))
	require.Equal(t, 1, len(cls))
	clID := cls[0].ID

	cases := []struct {
		desc  string
		token string
		ids   []string
		err   error
	}{
		{
			desc:  "remove clients with invalid ID",
			token: token,
			ids:   []string{wrongID},
			err:   dbutil.ErrNotFound,
		},
		{
			desc:  "remove clients with invalid token",
			token: wrongToken,
			ids:   []string{clID},
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "remove clients with valid token",
			token: token,
			ids:   []string{clID},
			err:   nil,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveClients(context.Background(), tc.token, tc.ids...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}

func TestRemoveClientsByThing(t *testing.T) {
	svc := newService()

	_, err := svc.CreateClients(context.Background(), token, thingID, client)
	require.Nil(t, err, fmt.Sprintf("unexpected error creating clients: %s", err))

	cases := []struct {
		desc    string
		thingID string
		err     error
	}{
		{
			desc:    "remove clients by thing with valid thing ID",
			thingID: thingID,
			err:     nil,
		},
		{
			desc:    "remove clients by thing with unknown thing ID",
			thingID: wrongID,
			err:     nil,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveClientsByThing(context.Background(), tc.thingID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}

func TestRemoveClientsByGroup(t *testing.T) {
	svc := newService()

	_, err := svc.CreateClients(context.Background(), token, thingID, client)
	require.Nil(t, err, fmt.Sprintf("unexpected error creating clients: %s", err))

	cases := []struct {
		desc    string
		groupID string
		err     error
	}{
		{
			desc:    "remove clients by group with valid group ID",
			groupID: groupID,
			err:     nil,
		},
		{
			desc:    "remove clients by group with unknown group ID",
			groupID: wrongID,
			err:     nil,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveClientsByGroup(context.Background(), tc.groupID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}
