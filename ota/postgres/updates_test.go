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
		Checksum:  "9f2ca71e",
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
			pm:    ota.PageMetadata{Order: "id"},
			size:  3,
			total: 3,
		},
		{
			desc:  "retrieve failed thing updates",
			pm:    ota.PageMetadata{Order: "id", Status: ota.StatusFailed},
			size:  1,
			total: 1,
		},
		{
			desc:  "retrieve page of thing updates",
			pm:    ota.PageMetadata{Order: "id", Limit: 2},
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

func TestUpdateThingUpdateNotFound(t *testing.T) {
	repo := postgres.NewUpdateRepository(dbutil.NewDatabase(db))

	err := repo.UpdateThingUpdate(context.Background(), ota.ThingUpdate{UpdateID: newID(t), ThingID: newID(t), Status: ota.StatusFailed, UpdatedAt: time.Now().UTC()})
	assert.True(t, errors.Contains(err, dbutil.ErrNotFound), fmt.Sprintf("expected %s got %s", dbutil.ErrNotFound, err))
}

func TestRetryFailed(t *testing.T) {
	repo := postgres.NewUpdateRepository(dbutil.NewDatabase(db))
	thingIDs := []string{newID(t), newID(t)}
	u := saveUpdate(t, repo, newID(t), time.Now().UTC(), thingIDs...)

	err := repo.UpdateThingUpdate(context.Background(), ota.ThingUpdate{UpdateID: u.ID, ThingID: thingIDs[0], Status: ota.StatusFailed, Error: "checksum mismatch", UpdatedAt: time.Now().UTC()})
	require.Nil(t, err, fmt.Sprintf("updating thing update failed: %s", err))

	before := time.Now().UTC()
	retried, err := repo.RetryFailed(context.Background(), u.ID)
	require.Nil(t, err, fmt.Sprintf("retrying failed thing updates failed: %s", err))
	assert.Equal(t, []string{thingIDs[0]}, retried, "only failed things must be retried")

	tu, err := repo.RetrieveThingUpdate(context.Background(), u.ID, thingIDs[0])
	require.Nil(t, err, fmt.Sprintf("retrieving thing update failed: %s", err))
	assert.Equal(t, ota.StatusPending, tu.Status, "retried thing update must be pending")
	assert.Empty(t, tu.Error, "retry must clear the error")
	assert.False(t, tu.UpdatedAt.Before(before.Truncate(time.Millisecond)), "retry must restart the timeout clock")
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
