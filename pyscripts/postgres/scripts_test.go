// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/pyscripts"
	"github.com/MainfluxLabs/mainflux/pyscripts/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var idp = uuid.New()

func cleanUp(t *testing.T) {
	_, err := db.Exec("TRUNCATE scripts, script_runs CASCADE")
	require.Nil(t, err, fmt.Sprintf("unexpected cleanup error: %s", err))
}

func newScript(t *testing.T, groupID, name, source string) pyscripts.Script {
	id, err := idp.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected id error: %s", err))

	return pyscripts.Script{
		ID:       id,
		GroupID:  groupID,
		Name:     name,
		Source:   source,
		Checksum: pyscripts.Checksum(source),
	}
}

func TestSaveScripts(t *testing.T) {
	cleanUp(t)
	repo := postgres.NewScriptRepository(dbutil.NewDatabase(db))
	ctx := context.Background()

	groupID, err := idp.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected id error: %s", err))

	s := newScript(t, groupID, "a", "1 + 1")

	cases := []struct {
		desc   string
		script pyscripts.Script
		err    error
	}{
		{
			desc:   "save a new script",
			script: s,
			err:    nil,
		},
		{
			desc:   "save a script with a conflicting id",
			script: s,
			err:    dbutil.ErrConflict,
		},
	}

	for _, tc := range cases {
		saved, err := repo.SaveScripts(ctx, tc.script)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
		if tc.err == nil {
			require.Len(t, saved, 1, tc.desc)
			assert.False(t, saved[0].CreatedAt.IsZero(), fmt.Sprintf("%s: expected created_at to be set", tc.desc))
		}
	}
}

func TestRetrieveScriptByID(t *testing.T) {
	cleanUp(t)
	repo := postgres.NewScriptRepository(dbutil.NewDatabase(db))
	ctx := context.Background()

	groupID, err := idp.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected id error: %s", err))

	s := newScript(t, groupID, "a", "1 + 1")
	_, err = repo.SaveScripts(ctx, s)
	require.Nil(t, err, fmt.Sprintf("unexpected save error: %s", err))

	nonexistent, err := idp.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected id error: %s", err))

	cases := []struct {
		desc string
		id   string
		err  error
	}{
		{
			desc: "retrieve an existing script",
			id:   s.ID,
			err:  nil,
		},
		{
			desc: "retrieve a non-existent script",
			id:   nonexistent,
			err:  dbutil.ErrNotFound,
		},
		{
			desc: "retrieve with a malformed id",
			id:   "not-a-uuid",
			err:  dbutil.ErrMalformedEntity,
		},
	}

	for _, tc := range cases {
		got, err := repo.RetrieveScriptByID(ctx, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
		if tc.err == nil {
			assert.Equal(t, s.ID, got.ID, fmt.Sprintf("%s: expected script %s", tc.desc, s.ID))
			assert.Equal(t, s.Checksum, got.Checksum, fmt.Sprintf("%s: expected checksum to round-trip", tc.desc))
		}
	}
}

func TestRetrieveScriptsByGroup(t *testing.T) {
	cleanUp(t)
	repo := postgres.NewScriptRepository(dbutil.NewDatabase(db))
	ctx := context.Background()

	groupID, err := idp.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected id error: %s", err))
	otherGroup, err := idp.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected id error: %s", err))

	for i := 0; i < 5; i++ {
		_, err := repo.SaveScripts(ctx, newScript(t, groupID, fmt.Sprintf("script-%d", i), "1 + 1"))
		require.Nil(t, err, fmt.Sprintf("unexpected save error: %s", err))
	}
	_, err = repo.SaveScripts(ctx, newScript(t, otherGroup, "elsewhere", "1 + 1"))
	require.Nil(t, err, fmt.Sprintf("unexpected save error: %s", err))

	cases := []struct {
		desc    string
		groupID string
		pm      pyscripts.PageMetadata
		total   uint64
		size    int
	}{
		{
			desc:    "retrieve all scripts in a group",
			groupID: groupID,
			pm:      pyscripts.PageMetadata{Limit: 10},
			total:   5,
			size:    5,
		},
		{
			desc:    "retrieve scripts with a limit",
			groupID: groupID,
			pm:      pyscripts.PageMetadata{Limit: 2},
			total:   5,
			size:    2,
		},
		{
			desc:    "retrieve scripts with an offset",
			groupID: groupID,
			pm:      pyscripts.PageMetadata{Offset: 4, Limit: 10},
			total:   5,
			size:    1,
		},
		{
			desc:    "retrieve scripts filtered by name",
			groupID: groupID,
			pm:      pyscripts.PageMetadata{Name: "script-1", Limit: 10},
			total:   1,
			size:    1,
		},
		{
			desc:    "retrieve scripts of another group",
			groupID: otherGroup,
			pm:      pyscripts.PageMetadata{Limit: 10},
			total:   1,
			size:    1,
		},
	}

	for _, tc := range cases {
		page, err := repo.RetrieveScriptsByGroup(ctx, tc.groupID, tc.pm)
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))
		assert.Equal(t, tc.total, page.Total, fmt.Sprintf("%s: expected total %d got %d", tc.desc, tc.total, page.Total))
		assert.Len(t, page.Scripts, tc.size, fmt.Sprintf("%s: expected %d scripts got %d", tc.desc, tc.size, len(page.Scripts)))
	}
}

func TestUpdateScript(t *testing.T) {
	cleanUp(t)
	repo := postgres.NewScriptRepository(dbutil.NewDatabase(db))
	ctx := context.Background()

	groupID, err := idp.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected id error: %s", err))

	s := newScript(t, groupID, "a", "1 + 1")
	_, err = repo.SaveScripts(ctx, s)
	require.Nil(t, err, fmt.Sprintf("unexpected save error: %s", err))

	updated := s
	updated.Name = "renamed"
	updated.Source = "2 + 2"
	updated.Checksum = pyscripts.Checksum(updated.Source)

	missing := newScript(t, groupID, "missing", "1 + 1")

	cases := []struct {
		desc   string
		script pyscripts.Script
		err    error
	}{
		{
			desc:   "update an existing script",
			script: updated,
			err:    nil,
		},
		{
			desc:   "update a non-existent script",
			script: missing,
			err:    dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := repo.UpdateScript(ctx, tc.script)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
		if tc.err == nil {
			got, err := repo.RetrieveScriptByID(ctx, tc.script.ID)
			require.Nil(t, err, fmt.Sprintf("%s: unexpected retrieve error: %s", tc.desc, err))
			assert.Equal(t, tc.script.Source, got.Source, fmt.Sprintf("%s: expected updated source", tc.desc))
			assert.Equal(t, tc.script.Checksum, got.Checksum, fmt.Sprintf("%s: expected updated checksum", tc.desc))
		}
	}
}

func TestRemoveScripts(t *testing.T) {
	cleanUp(t)
	repo := postgres.NewScriptRepository(dbutil.NewDatabase(db))
	ctx := context.Background()

	groupID, err := idp.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected id error: %s", err))

	s := newScript(t, groupID, "a", "1 + 1")
	_, err = repo.SaveScripts(ctx, s)
	require.Nil(t, err, fmt.Sprintf("unexpected save error: %s", err))

	err = repo.RemoveScripts(ctx, s.ID)
	require.Nil(t, err, fmt.Sprintf("unexpected remove error: %s", err))

	_, err = repo.RetrieveScriptByID(ctx, s.ID)
	assert.True(t, errors.Contains(err, dbutil.ErrNotFound), "expected script to be removed")

	// Removing a non-existent id is a no-op, not an error.
	err = repo.RemoveScripts(ctx, s.ID)
	assert.Nil(t, err, "expected removing a missing script to be a no-op")
}

func TestRemoveScriptsByGroup(t *testing.T) {
	cleanUp(t)
	repo := postgres.NewScriptRepository(dbutil.NewDatabase(db))
	ctx := context.Background()

	groupID, err := idp.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected id error: %s", err))
	otherGroup, err := idp.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected id error: %s", err))

	for i := 0; i < 3; i++ {
		_, err := repo.SaveScripts(ctx, newScript(t, groupID, fmt.Sprintf("s-%d", i), "1 + 1"))
		require.Nil(t, err, fmt.Sprintf("unexpected save error: %s", err))
	}
	keep := newScript(t, otherGroup, "keep", "1 + 1")
	_, err = repo.SaveScripts(ctx, keep)
	require.Nil(t, err, fmt.Sprintf("unexpected save error: %s", err))

	err = repo.RemoveScriptsByGroup(ctx, groupID)
	require.Nil(t, err, fmt.Sprintf("unexpected remove-by-group error: %s", err))

	page, err := repo.RetrieveScriptsByGroup(ctx, groupID, pyscripts.PageMetadata{Limit: 10})
	require.Nil(t, err, fmt.Sprintf("unexpected retrieve error: %s", err))
	assert.Equal(t, uint64(0), page.Total, "expected the group's scripts to be removed")

	// Another group's scripts are untouched.
	page, err = repo.RetrieveScriptsByGroup(ctx, otherGroup, pyscripts.PageMetadata{Limit: 10})
	require.Nil(t, err, fmt.Sprintf("unexpected retrieve error: %s", err))
	assert.Equal(t, uint64(1), page.Total, "expected other group's scripts to remain")
}
