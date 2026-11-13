// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/ota"
	"github.com/MainfluxLabs/mainflux/ota/postgres"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var idProvider = uuid.New()

func newID(t *testing.T) string {
	t.Helper()

	id, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("generating id failed: %s", err))

	return id
}

func saveUpdate(t *testing.T, repo ota.UpdateRepository, groupID string, createdAt time.Time, thingIDs ...string) ota.Update {
	t.Helper()

	u := ota.Update{
		ID:        newID(t),
		GroupID:   groupID,
		FileName:  "fw.bin",
		Version:   "1.4.0",
		FileSize:  412,
		Checksum:  "9f2c4a0e7b1d3c5f6a8e9b0c2d4f6a8b0c2e4f6a8b0c2d4e6f8a0b2c4d6ea71e",
		Timeout:   time.Hour,
		CreatedAt: createdAt,
	}

	err := repo.Save(context.Background(), u, thingIDs...)
	require.Nil(t, err, fmt.Sprintf("saving update failed: %s", err))

	return u
}

func TestSaveSupersedes(t *testing.T) {
	repo := postgres.NewUpdateRepository(dbutil.NewDatabase(db))
	groupID, thingID := newID(t), newID(t)
	now := time.Now().UTC().Truncate(time.Millisecond)

	first := saveUpdate(t, repo, groupID, now, thingID)
	second := saveUpdate(t, repo, groupID, now.Add(time.Second), thingID)

	old, err := repo.RetrieveThingUpdate(context.Background(), first.ID, thingID)
	require.Nil(t, err, fmt.Sprintf("retrieving thing update failed: %s", err))
	assert.Equal(t, ota.StatusFailed, old.Status, "older active thing update must fail")
	assert.Equal(t, fmt.Sprintf(ota.ErrMsgSuperseded, second.ID), old.Error, "superseded error mismatch")

	active, err := repo.RetrieveActiveThingUpdate(context.Background(), thingID)
	require.Nil(t, err, fmt.Sprintf("retrieving active thing update failed: %s", err))
	assert.Equal(t, second.ID, active.UpdateID, "active thing update must belong to the newest update")
}

func TestRetrieveByID(t *testing.T) {
	repo := postgres.NewUpdateRepository(dbutil.NewDatabase(db))
	groupID := newID(t)
	thingIDs := []string{newID(t), newID(t), newID(t)}
	u := saveUpdate(t, repo, groupID, time.Now().UTC(), thingIDs...)

	err := repo.UpdateThingUpdate(context.Background(), ota.ThingUpdate{UpdateID: u.ID, ThingID: thingIDs[0], Status: ota.StatusDownloading, UpdatedAt: time.Now().UTC()})
	require.Nil(t, err, fmt.Sprintf("updating thing update failed: %s", err))
	err = repo.UpdateThingUpdate(context.Background(), ota.ThingUpdate{UpdateID: u.ID, ThingID: thingIDs[1], Status: ota.StatusSucceeded, ReportedVersion: "1.4.0", UpdatedAt: time.Now().UTC()})
	require.Nil(t, err, fmt.Sprintf("updating thing update failed: %s", err))

	cases := []struct {
		desc  string
		id    string
		stats ota.UpdateStats
		err   error
	}{
		{
			desc:  "retrieve update with stats",
			id:    u.ID,
			stats: ota.UpdateStats{Targets: 3, Pending: 1, Downloading: 1, Succeeded: 1},
			err:   nil,
		},
		{
			desc: "retrieve non-existing update",
			id:   newID(t),
			err:  dbutil.ErrNotFound,
		},
		{
			desc: "retrieve update with invalid id",
			id:   "invalid",
			err:  dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		got, err := repo.RetrieveByID(context.Background(), tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
		if tc.err != nil {
			continue
		}
		assert.Equal(t, tc.stats, got.Stats, fmt.Sprintf("%s: stats mismatch", tc.desc))
		assert.Equal(t, time.Hour, got.Timeout, fmt.Sprintf("%s: timeout mismatch", tc.desc))
	}
}

func TestRetrieveByGroup(t *testing.T) {
	repo := postgres.NewUpdateRepository(dbutil.NewDatabase(db))
	groupID := newID(t)
	now := time.Now().UTC()
	for i := range 3 {
		saveUpdate(t, repo, groupID, now.Add(time.Duration(i)*time.Second), newID(t))
	}

	cases := []struct {
		desc    string
		groupID string
		pm      ota.PageMetadata
		size    int
		total   uint64
		err     error
	}{
		{
			desc:    "retrieve all updates of group",
			groupID: groupID,
			pm:      ota.PageMetadata{Order: "created_at"},
			size:    3,
			total:   3,
		},
		{
			desc:    "retrieve page of updates",
			groupID: groupID,
			pm:      ota.PageMetadata{Offset: 1, Limit: 1, Order: "created_at"},
			size:    1,
			total:   3,
		},
		{
			desc:    "retrieve updates of empty group",
			groupID: newID(t),
			pm:      ota.PageMetadata{},
			size:    0,
			total:   0,
		},
		{
			desc:    "retrieve updates with invalid group id",
			groupID: "invalid",
			pm:      ota.PageMetadata{},
			err:     dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		page, err := repo.RetrieveByGroup(context.Background(), tc.groupID, tc.pm)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
		assert.Len(t, page.Updates, tc.size, fmt.Sprintf("%s: size mismatch", tc.desc))
		assert.Equal(t, tc.total, page.Total, fmt.Sprintf("%s: total mismatch", tc.desc))
	}
}

func TestRetrieveThingUpdates(t *testing.T) {
	repo := postgres.NewUpdateRepository(dbutil.NewDatabase(db))
	thingIDs := []string{newID(t), newID(t), newID(t)}
	u := saveUpdate(t, repo, newID(t), time.Now().UTC(), thingIDs...)

	err := repo.UpdateThingUpdate(context.Background(), ota.ThingUpdate{UpdateID: u.ID, ThingID: thingIDs[0], Status: ota.StatusFailed, Error: "checksum mismatch", UpdatedAt: time.Now().UTC()})
	require.Nil(t, err, fmt.Sprintf("updating thing update failed: %s", err))

	cases := []struct {
		desc  string
		pm    ota.PageMetadata
		size  int
		total uint64
	}{
		{
			desc:  "retrieve all thing updates",
			pm:    ota.PageMetadata{Order: "thing_id"},
			size:  3,
			total: 3,
		},
		{
			desc:  "retrieve failed thing updates",
			pm:    ota.PageMetadata{Order: "thing_id", Status: ota.StatusFailed},
			size:  1,
			total: 1,
		},
		{
			desc:  "retrieve page of thing updates",
			pm:    ota.PageMetadata{Order: "thing_id", Limit: 2},
			size:  2,
			total: 3,
		},
	}

	for _, tc := range cases {
		page, err := repo.RetrieveThingUpdates(context.Background(), u.ID, tc.pm)
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))
		assert.Len(t, page.ThingUpdates, tc.size, fmt.Sprintf("%s: size mismatch", tc.desc))
		assert.Equal(t, tc.total, page.Total, fmt.Sprintf("%s: total mismatch", tc.desc))
	}
}

func TestUpdateThingUpdate(t *testing.T) {
	repo := postgres.NewUpdateRepository(dbutil.NewDatabase(db))

	cases := []struct {
		desc   string
		setup  func(t *testing.T) (updateID, thingID string)
		status string
		err    error
	}{
		{
			desc: "update non-existing thing update",
			setup: func(t *testing.T) (string, string) {
				return newID(t), newID(t)
			},
			err: dbutil.ErrNotFound,
		},
		{
			desc: "update active thing update",
			setup: func(t *testing.T) (string, string) {
				thingID := newID(t)
				u := saveUpdate(t, repo, newID(t), time.Now().UTC(), thingID)
				return u.ID, thingID
			},
			status: ota.StatusDownloading,
		},
		{
			desc: "update timed out thing update",
			setup: func(t *testing.T) (string, string) {
				thingID := newID(t)
				u := saveUpdate(t, repo, newID(t), time.Now().UTC(), thingID)
				err := repo.FailTimedOut(context.Background(), time.Now().UTC().Add(48*time.Hour))
				require.Nil(t, err, fmt.Sprintf("failing timed out thing updates failed: %s", err))
				return u.ID, thingID
			},
			status: ota.StatusFailed,
			err:    dbutil.ErrNotFound,
		},
		{
			desc: "update superseded thing update",
			setup: func(t *testing.T) (string, string) {
				groupID, thingID := newID(t), newID(t)
				now := time.Now().UTC()
				u := saveUpdate(t, repo, groupID, now, thingID)
				saveUpdate(t, repo, groupID, now.Add(time.Second), thingID)
				return u.ID, thingID
			},
			status: ota.StatusFailed,
			err:    dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		updateID, thingID := tc.setup(t)

		err := repo.UpdateThingUpdate(context.Background(), ota.ThingUpdate{UpdateID: updateID, ThingID: thingID, Status: ota.StatusDownloading, UpdatedAt: time.Now().UTC()})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))

		if tc.status == "" {
			continue
		}

		tu, err := repo.RetrieveThingUpdate(context.Background(), updateID, thingID)
		require.Nil(t, err, fmt.Sprintf("%s: retrieving thing update failed: %s", tc.desc, err))
		assert.Equal(t, tc.status, tu.Status, fmt.Sprintf("%s: status mismatch", tc.desc))
	}
}

func TestRetryFailed(t *testing.T) {
	repo := postgres.NewUpdateRepository(dbutil.NewDatabase(db))

	cases := []struct {
		desc        string
		newerStatus string
		retried     bool
	}{
		{
			desc:    "retry failed thing update",
			retried: true,
		},
		{
			desc:        "retry thing update superseded by an active update",
			newerStatus: ota.StatusPending,
		},
		{
			desc:        "retry thing update superseded by a finished update",
			newerStatus: ota.StatusSucceeded,
		},
	}

	for _, tc := range cases {
		groupID, thingID, otherThingID := newID(t), newID(t), newID(t)
		now := time.Now().UTC()
		u := saveUpdate(t, repo, groupID, now, thingID, otherThingID)

		change := ota.ThingUpdate{UpdateID: u.ID, ThingID: thingID, Status: ota.StatusFailed, Error: "checksum mismatch", UpdatedAt: now}
		if tc.newerStatus != "" {
			newer := saveUpdate(t, repo, groupID, now.Add(time.Second), thingID)
			change = ota.ThingUpdate{UpdateID: newer.ID, ThingID: thingID, Status: tc.newerStatus, UpdatedAt: now}
		}
		err := repo.UpdateThingUpdate(context.Background(), change)
		require.Nil(t, err, fmt.Sprintf("%s: updating thing update failed: %s", tc.desc, err))

		before := time.Now().UTC()
		retried, err := repo.RetryFailed(context.Background(), u.ID)
		require.Nil(t, err, fmt.Sprintf("%s: retrying failed thing updates failed: %s", tc.desc, err))

		tu, err := repo.RetrieveThingUpdate(context.Background(), u.ID, thingID)
		require.Nil(t, err, fmt.Sprintf("%s: retrieving thing update failed: %s", tc.desc, err))

		if !tc.retried {
			assert.Empty(t, retried, fmt.Sprintf("%s: things with a newer update must not be retried", tc.desc))
			assert.Equal(t, ota.StatusFailed, tu.Status, fmt.Sprintf("%s: superseded thing update must stay failed", tc.desc))
			continue
		}

		assert.Equal(t, []string{thingID}, retried, fmt.Sprintf("%s: only failed things must be retried", tc.desc))
		assert.Equal(t, ota.StatusPending, tu.Status, fmt.Sprintf("%s: retried thing update must be pending", tc.desc))
		assert.Empty(t, tu.Error, fmt.Sprintf("%s: retry must clear the error", tc.desc))
		assert.False(t, tu.UpdatedAt.Before(before.Truncate(time.Millisecond)), fmt.Sprintf("%s: retry must restart the timeout clock", tc.desc))
	}
}

func TestOneActiveUpdate(t *testing.T) {
	repo := postgres.NewUpdateRepository(dbutil.NewDatabase(db))
	groupID, thingID := newID(t), newID(t)
	now := time.Now().UTC()

	first := saveUpdate(t, repo, groupID, now, thingID)
	saveUpdate(t, repo, groupID, now.Add(time.Second), thingID)

	// Written directly, since UpdateThingUpdate never reopens a finished thing update.
	_, err := db.Exec(`UPDATE thing_updates SET status = $1 WHERE update_id = $2 AND thing_id = $3`, ota.StatusPending, first.ID, thingID)
	pgErr, ok := err.(*pgconn.PgError)
	require.True(t, ok, fmt.Sprintf("expected a postgres error, got %v", err))
	assert.Equal(t, pgerrcode.UniqueViolation, pgErr.Code, "a second active update for a thing must be rejected")
}

func TestFailTimedOut(t *testing.T) {
	repo := postgres.NewUpdateRepository(dbutil.NewDatabase(db))
	thingID := newID(t)
	past := time.Now().UTC().Add(-2 * time.Hour)
	u := saveUpdate(t, repo, newID(t), past, thingID)

	err := repo.FailTimedOut(context.Background(), time.Now().UTC())
	require.Nil(t, err, fmt.Sprintf("failing timed out thing updates failed: %s", err))

	tu, err := repo.RetrieveThingUpdate(context.Background(), u.ID, thingID)
	require.Nil(t, err, fmt.Sprintf("retrieving thing update failed: %s", err))
	assert.Equal(t, ota.StatusFailed, tu.Status, "expired thing update must fail")
	assert.Equal(t, ota.ErrMsgTimedOut, tu.Error, "error mismatch")
}

func TestRemove(t *testing.T) {
	repo := postgres.NewUpdateRepository(dbutil.NewDatabase(db))
	groupID, thingID := newID(t), newID(t)
	u := saveUpdate(t, repo, groupID, time.Now().UTC(), thingID)
	byGroup := saveUpdate(t, repo, groupID, time.Now().UTC(), newID(t))

	err := repo.Remove(context.Background(), u.ID)
	require.Nil(t, err, fmt.Sprintf("removing update failed: %s", err))

	_, err = repo.RetrieveThingUpdate(context.Background(), u.ID, thingID)
	assert.True(t, errors.Contains(err, dbutil.ErrNotFound), fmt.Sprintf("thing updates must be removed with the update: got %s", err))

	err = repo.RemoveByGroup(context.Background(), groupID)
	require.Nil(t, err, fmt.Sprintf("removing updates by group failed: %s", err))

	_, err = repo.RetrieveByID(context.Background(), byGroup.ID)
	assert.True(t, errors.Contains(err, dbutil.ErrNotFound), fmt.Sprintf("expected %s got %s", dbutil.ErrNotFound, err))
}

func TestRemoveThingUpdatesByThing(t *testing.T) {
	repo := postgres.NewUpdateRepository(dbutil.NewDatabase(db))
	thingID := newID(t)
	u := saveUpdate(t, repo, newID(t), time.Now().UTC(), thingID, newID(t))

	err := repo.RemoveThingUpdatesByThing(context.Background(), thingID)
	require.Nil(t, err, fmt.Sprintf("removing thing updates failed: %s", err))

	got, err := repo.RetrieveByID(context.Background(), u.ID)
	require.Nil(t, err, fmt.Sprintf("retrieving update failed: %s", err))
	assert.Equal(t, uint64(1), got.Stats.Targets, "removed thing must not be counted")
}
