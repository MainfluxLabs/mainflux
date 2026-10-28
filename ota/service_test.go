// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package ota_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/ota"
	otamocks "github.com/MainfluxLabs/mainflux/ota/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging/nats"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	token        = "admin@example.com"
	wrongValue   = "wrong-value"
	thingID      = "5384fb1c-d0ae-4cbe-be52-c54223150fe0"
	otherThingID = "6384fb1c-d0ae-4cbe-be52-c54223150fe1"
	groupID      = "574106f7-030e-4881-8ab0-151195c29f94"
	thingKey     = "8384fb1c-d0ae-4cbe-be52-c54223150fe2"
	fwVersion    = "1.4.0"
	firmwareURL  = "https://localhost/fs/groupfiles"
	otaSubtopic  = "ota"
)

var firmware = ota.Update{
	FileName: "fw-sensor-1.4.0.bin",
	Version:  fwVersion,
	FileSize: 412,
	Checksum: "9f2ca71e",
	Timeout:  time.Hour,
}

func newService() (ota.Service, *otamocks.CommandPublisher) {
	things := map[string]domain.Thing{
		token:    {ID: thingID, GroupID: groupID},
		thingKey: {ID: thingID, GroupID: groupID},
	}
	groups := map[string]domain.Group{token: {ID: groupID}}
	tc := mocks.NewThingsServiceClient(nil, things, groups)
	pub := otamocks.NewCommandPublisher()

	return ota.New(tc, otamocks.NewUpdateRepository(), pub, uuid.NewMock(), firmwareURL, logger.NewMock()), pub
}

func createUpdate(t *testing.T, svc ota.Service, u ota.Update) ota.Update {
	t.Helper()

	created, err := svc.CreateUpdate(context.Background(), token, groupID, u, thingID)
	require.Nil(t, err, fmt.Sprintf("creating update failed: %s", err))

	return created
}

func key() domain.ThingKey {
	return domain.ThingKey{Value: thingKey, Type: domain.KeyTypeInternal}
}

func TestCreateUpdate(t *testing.T) {
	svc, pub := newService()

	cases := []struct {
		desc     string
		token    string
		thingIDs []string
		err      error
	}{
		{
			desc:     "create update",
			token:    token,
			thingIDs: []string{thingID},
			err:      nil,
		},
		{
			desc:     "create update without things",
			token:    token,
			thingIDs: []string{},
			err:      errors.ErrMalformedEntity,
		},
		{
			desc:     "create update with wrong credentials",
			token:    wrongValue,
			thingIDs: []string{thingID},
			err:      errors.ErrAuthentication,
		},
		{
			desc:     "create update for thing the user cannot access",
			token:    token,
			thingIDs: []string{otherThingID},
			err:      errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		u, err := svc.CreateUpdate(context.Background(), tc.token, groupID, firmware, tc.thingIDs...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
		if tc.err != nil {
			continue
		}

		assert.Equal(t, groupID, u.GroupID, fmt.Sprintf("%s: group mismatch", tc.desc))
		assert.Equal(t, ota.UpdateStats{Targets: 1, Pending: 1}, u.Stats, fmt.Sprintf("%s: stats mismatch", tc.desc))

		cmds := pub.Commands(nats.GetThingCommandsSubject(thingID, otaSubtopic))
		require.Len(t, cmds, 1, fmt.Sprintf("%s: expected one ota command", tc.desc))

		var payload map[string]any
		require.Nil(t, json.Unmarshal(cmds[0].Payload, &payload), fmt.Sprintf("%s: invalid command payload", tc.desc))
		assert.Equal(t, u.ID, payload["update_id"], fmt.Sprintf("%s: command update_id mismatch", tc.desc))
		assert.Equal(t, fwVersion, payload["version"], fmt.Sprintf("%s: command version mismatch", tc.desc))
		assert.Equal(t, firmwareURL+"/"+firmware.FileName, payload["url"], fmt.Sprintf("%s: command url mismatch", tc.desc))
	}
}

func TestCreateUpdateSupersedes(t *testing.T) {
	svc, _ := newService()
	first := createUpdate(t, svc, firmware)
	second := createUpdate(t, svc, firmware)

	page, err := svc.ListThingUpdates(context.Background(), token, first.ID, ota.PageMetadata{})
	require.Nil(t, err, fmt.Sprintf("listing thing updates failed: %s", err))
	require.Len(t, page.ThingUpdates, 1, "expected one thing update")
	assert.Equal(t, ota.StatusFailed, page.ThingUpdates[0].Status, "older update must fail when superseded")
	assert.Equal(t, "superseded by update "+second.ID, page.ThingUpdates[0].Error, "superseded error mismatch")

	pending, err := svc.ViewPending(context.Background(), key())
	require.Nil(t, err, fmt.Sprintf("viewing pending update failed: %s", err))
	assert.Equal(t, second.ID, pending.UpdateID, "pending update must be the newest one")
}

func TestViewUpdate(t *testing.T) {
	svc, _ := newService()
	u := createUpdate(t, svc, firmware)

	cases := []struct {
		desc  string
		token string
		id    string
		err   error
	}{
		{
			desc:  "view update",
			token: token,
			id:    u.ID,
			err:   nil,
		},
		{
			desc:  "view update with wrong credentials",
			token: wrongValue,
			id:    u.ID,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "view non-existing update",
			token: token,
			id:    wrongValue,
			err:   dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		got, err := svc.ViewUpdate(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
		if tc.err == nil {
			assert.Equal(t, fwVersion, got.Version, fmt.Sprintf("%s: version mismatch", tc.desc))
		}
	}
}

func TestListUpdatesByGroup(t *testing.T) {
	svc, _ := newService()
	for range 3 {
		createUpdate(t, svc, firmware)
	}

	cases := []struct {
		desc  string
		token string
		pm    ota.PageMetadata
		size  int
		err   error
	}{
		{
			desc:  "list all updates",
			token: token,
			pm:    ota.PageMetadata{},
			size:  3,
			err:   nil,
		},
		{
			desc:  "list updates with limit",
			token: token,
			pm:    ota.PageMetadata{Limit: 2},
			size:  2,
			err:   nil,
		},
		{
			desc:  "list updates with wrong credentials",
			token: wrongValue,
			pm:    ota.PageMetadata{},
			size:  0,
			err:   errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		page, err := svc.ListUpdatesByGroup(context.Background(), tc.token, groupID, tc.pm)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
		assert.Len(t, page.Updates, tc.size, fmt.Sprintf("%s: size mismatch", tc.desc))
	}
}

func TestReportStatus(t *testing.T) {
	svc, _ := newService()
	u := createUpdate(t, svc, firmware)

	cases := []struct {
		desc   string
		key    domain.ThingKey
		report ota.StatusReport
		status string
		err    error
	}{
		{
			desc:   "report downloading",
			key:    key(),
			report: ota.StatusReport{UpdateID: u.ID, Status: ota.StatusDownloading},
			status: ota.StatusDownloading,
			err:    nil,
		},
		{
			desc:   "report installing",
			key:    key(),
			report: ota.StatusReport{UpdateID: u.ID, Status: ota.StatusInstalling},
			status: ota.StatusInstalling,
			err:    nil,
		},
		{
			desc:   "report invalid status",
			key:    key(),
			report: ota.StatusReport{UpdateID: u.ID, Status: wrongValue},
			status: ota.StatusInstalling,
			err:    ota.ErrInvalidStatus,
		},
		{
			desc:   "report with wrong key",
			key:    domain.ThingKey{Value: wrongValue, Type: domain.KeyTypeInternal},
			report: ota.StatusReport{UpdateID: u.ID, Status: ota.StatusDownloading},
			status: ota.StatusInstalling,
			err:    errors.ErrAuthentication,
		},
		{
			desc:   "report for unknown update",
			key:    key(),
			report: ota.StatusReport{UpdateID: wrongValue, Status: ota.StatusDownloading},
			status: ota.StatusInstalling,
			err:    dbutil.ErrNotFound,
		},
		{
			desc:   "report succeeded",
			key:    key(),
			report: ota.StatusReport{UpdateID: u.ID, Status: ota.StatusSucceeded, Version: fwVersion},
			status: ota.StatusSucceeded,
			err:    nil,
		},
		{
			desc:   "report after update finished",
			key:    key(),
			report: ota.StatusReport{UpdateID: u.ID, Status: ota.StatusFailed},
			status: ota.StatusSucceeded,
			err:    ota.ErrUpdateFinished,
		},
	}

	for _, tc := range cases {
		err := svc.ReportStatus(context.Background(), tc.key, tc.report)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))

		page, err := svc.ListThingUpdates(context.Background(), token, u.ID, ota.PageMetadata{})
		require.Nil(t, err, fmt.Sprintf("%s: listing thing updates failed: %s", tc.desc, err))
		assert.Equal(t, tc.status, page.ThingUpdates[0].Status, fmt.Sprintf("%s: status mismatch", tc.desc))
	}
}

func TestReportStatusVersionMismatch(t *testing.T) {
	svc, _ := newService()
	u := createUpdate(t, svc, firmware)

	err := svc.ReportStatus(context.Background(), key(), ota.StatusReport{UpdateID: u.ID, Status: ota.StatusSucceeded, Version: "1.3.0"})
	require.Nil(t, err, fmt.Sprintf("reporting status failed: %s", err))

	page, err := svc.ListThingUpdates(context.Background(), token, u.ID, ota.PageMetadata{})
	require.Nil(t, err, fmt.Sprintf("listing thing updates failed: %s", err))
	assert.Equal(t, ota.StatusFailed, page.ThingUpdates[0].Status, "wrong reported version must fail the update")
	assert.Equal(t, "version mismatch: reported 1.3.0", page.ThingUpdates[0].Error, "error mismatch")
	assert.Equal(t, "1.3.0", page.ThingUpdates[0].ReportedVersion, "reported version mismatch")
}

func TestViewPending(t *testing.T) {
	svc, _ := newService()

	_, err := svc.ViewPending(context.Background(), key())
	assert.True(t, errors.Contains(err, dbutil.ErrNotFound), fmt.Sprintf("expected %s got %s", dbutil.ErrNotFound, err))

	u := createUpdate(t, svc, firmware)
	pending, err := svc.ViewPending(context.Background(), key())
	require.Nil(t, err, fmt.Sprintf("viewing pending update failed: %s", err))
	assert.Equal(t, u.ID, pending.UpdateID, "pending update mismatch")
	assert.Equal(t, firmwareURL+"/"+firmware.FileName, pending.URL, "pending url mismatch")

	err = svc.ReportStatus(context.Background(), key(), ota.StatusReport{UpdateID: u.ID, Status: ota.StatusSucceeded, Version: fwVersion})
	require.Nil(t, err, fmt.Sprintf("reporting status failed: %s", err))

	_, err = svc.ViewPending(context.Background(), key())
	assert.True(t, errors.Contains(err, dbutil.ErrNotFound), fmt.Sprintf("finished update must not be pending: got %s", err))
}

func TestRetryFailed(t *testing.T) {
	svc, pub := newService()
	u := createUpdate(t, svc, firmware)

	err := svc.ReportStatus(context.Background(), key(), ota.StatusReport{UpdateID: u.ID, Status: ota.StatusFailed, Error: "checksum mismatch"})
	require.Nil(t, err, fmt.Sprintf("reporting status failed: %s", err))

	err = svc.RetryFailed(context.Background(), wrongValue, u.ID)
	assert.True(t, errors.Contains(err, errors.ErrAuthentication), fmt.Sprintf("expected %s got %s", errors.ErrAuthentication, err))

	err = svc.RetryFailed(context.Background(), token, u.ID)
	require.Nil(t, err, fmt.Sprintf("retrying failed update failed: %s", err))

	page, err := svc.ListThingUpdates(context.Background(), token, u.ID, ota.PageMetadata{})
	require.Nil(t, err, fmt.Sprintf("listing thing updates failed: %s", err))
	assert.Equal(t, ota.StatusPending, page.ThingUpdates[0].Status, "retried thing must be pending")
	assert.Empty(t, page.ThingUpdates[0].Error, "retry must clear the error")
	assert.Len(t, pub.Commands(nats.GetThingCommandsSubject(thingID, otaSubtopic)), 2, "retry must send the command again")
}

func TestFailTimedOut(t *testing.T) {
	svc, _ := newService()
	expired := firmware
	expired.Timeout = time.Nanosecond
	u := createUpdate(t, svc, expired)

	err := svc.FailTimedOut(context.Background())
	require.Nil(t, err, fmt.Sprintf("failing timed out updates failed: %s", err))

	page, err := svc.ListThingUpdates(context.Background(), token, u.ID, ota.PageMetadata{})
	require.Nil(t, err, fmt.Sprintf("listing thing updates failed: %s", err))
	assert.Equal(t, ota.StatusFailed, page.ThingUpdates[0].Status, "expired thing update must fail")
	assert.Equal(t, "timed out", page.ThingUpdates[0].Error, "error mismatch")
}

func TestConsumeMessage(t *testing.T) {
	svc, _ := newService()
	u := createUpdate(t, svc, firmware)

	cases := []struct {
		desc    string
		payload []byte
		status  string
		err     error
	}{
		{
			desc:    "consume status report",
			payload: []byte(fmt.Sprintf(`{"update_id":"%s","status":"downloading"}`, u.ID)),
			status:  ota.StatusDownloading,
			err:     nil,
		},
		{
			desc:    "consume malformed status report",
			payload: []byte(`{"update_id":`),
			status:  ota.StatusDownloading,
			err:     errors.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		err := svc.ConsumeMessage(nats.SubjectOTA, protomfx.Message{ThingId: thingID, Subtopic: otaSubtopic, Payload: tc.payload})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))

		page, err := svc.ListThingUpdates(context.Background(), token, u.ID, ota.PageMetadata{})
		require.Nil(t, err, fmt.Sprintf("%s: listing thing updates failed: %s", tc.desc, err))
		assert.Equal(t, tc.status, page.ThingUpdates[0].Status, fmt.Sprintf("%s: status mismatch", tc.desc))
	}
}
