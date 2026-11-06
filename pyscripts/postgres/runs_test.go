// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pyscripts"
	"github.com/MainfluxLabs/mainflux/pyscripts/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func saveScriptFixture(t *testing.T, repo pyscripts.ScriptRepository) pyscripts.Script {
	groupID, err := idp.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected id error: %s", err))

	s := newScript(t, groupID, "fixture", "1 + 1")
	_, err = repo.SaveScripts(context.Background(), s)
	require.Nil(t, err, fmt.Sprintf("unexpected save error: %s", err))

	return s
}

func newRun(t *testing.T, scriptID, status string, value any, logs []string) pyscripts.ScriptRun {
	id, err := idp.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected id error: %s", err))

	return pyscripts.ScriptRun{
		ID:             id,
		ScriptID:       scriptID,
		ScriptChecksum: pyscripts.Checksum("1 + 1"),
		Value:          value,
		Logs:           logs,
		StartedAt:      time.Now(),
		FinishedAt:     time.Now(),
		Status:         status,
	}
}

func TestSaveRuns(t *testing.T) {
	cleanUp(t)
	repo := postgres.NewScriptRepository(dbutil.NewDatabase(db))
	ctx := context.Background()

	s := saveScriptFixture(t, repo)

	unknownScript, err := idp.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected id error: %s", err))

	cases := []struct {
		desc string
		run  pyscripts.ScriptRun
		err  error
	}{
		{
			desc: "save a run for an existing script",
			run:  newRun(t, s.ID, pyscripts.ScriptRunStatusSuccess, float64(2), []string{"log line"}),
			err:  nil,
		},
		{
			desc: "save a run for an unknown script",
			run:  newRun(t, unknownScript, pyscripts.ScriptRunStatusSuccess, float64(2), nil),
			err:  dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		saved, err := repo.SaveRuns(ctx, tc.run)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
		if tc.err == nil {
			require.Len(t, saved, 1, tc.desc)
		}
	}
}

func TestSaveRunsRetention(t *testing.T) {
	cleanUp(t)
	repo := postgres.NewScriptRepository(dbutil.NewDatabase(db))
	ctx := context.Background()

	s := saveScriptFixture(t, repo)

	// cap mirrors maxRunsPerScript in runs.go; this test is the guard on it. Save
	// a few past the cap, each newer than the last, one at a time as the service
	// does: the newest cap must survive and the oldest beyond it must be pruned.
	const cap, extra = 1000, 5
	base := time.Now().Add(-2 * time.Hour)
	var ids []string
	for i := 0; i < cap+extra; i++ {
		run := newRun(t, s.ID, pyscripts.ScriptRunStatusSuccess, float64(i), nil)
		run.StartedAt = base.Add(time.Duration(i) * time.Second)
		run.FinishedAt = run.StartedAt
		_, err := repo.SaveRuns(ctx, run)
		require.Nil(t, err, fmt.Sprintf("unexpected save error on run %d: %s", i, err))
		ids = append(ids, run.ID)
	}

	page, err := repo.RetrieveRunsByScript(ctx, s.ID, pyscripts.PageMetadata{Limit: 1})
	require.Nil(t, err, fmt.Sprintf("unexpected retrieve error: %s", err))
	assert.Equal(t, uint64(cap), page.Total, "expected the run history to be capped")

	// The oldest beyond the cap are gone; the newest is retained.
	for _, oldID := range ids[:extra] {
		_, err := repo.RetrieveRunByID(ctx, oldID)
		assert.True(t, errors.Contains(err, dbutil.ErrNotFound), fmt.Sprintf("expected pruned run %s to be gone, got %s", oldID, err))
	}
	_, err = repo.RetrieveRunByID(ctx, ids[len(ids)-1])
	assert.Nil(t, err, "expected the newest run to be retained")
}

func TestSaveRunWithNulByte(t *testing.T) {
	cleanUp(t)
	repo := postgres.NewScriptRepository(dbutil.NewDatabase(db))
	ctx := context.Background()

	s := saveScriptFixture(t, repo)

	run := newRun(t, s.ID, pyscripts.ScriptRunStatusSuccess, "x\u0000y", []string{"log\u0000line"})

	_, err := repo.SaveRuns(ctx, run)
	require.Nil(t, err, fmt.Sprintf("unexpected save error for NUL byte payload: %s", err))

	got, err := repo.RetrieveRunByID(ctx, run.ID)
	require.Nil(t, err, fmt.Sprintf("unexpected retrieve error: %s", err))
	assert.Equal(t, "x\u0000y", got.Value, "expected a NUL byte in the value to round-trip")
	require.Len(t, got.Logs, 1, "expected one log line")
	assert.Equal(t, "log\u0000line", got.Logs[0], "expected a NUL byte in a log line to round-trip")
}

func TestRetrieveRunByID(t *testing.T) {
	cleanUp(t)
	repo := postgres.NewScriptRepository(dbutil.NewDatabase(db))
	ctx := context.Background()

	s := saveScriptFixture(t, repo)

	run := newRun(t, s.ID, pyscripts.ScriptRunStatusFail, nil, nil)
	run.Error = "boom"
	_, err := repo.SaveRuns(ctx, run)
	require.Nil(t, err, fmt.Sprintf("unexpected save error: %s", err))

	nonexistent, err := idp.ID()
	require.Nil(t, err, fmt.Sprintf("unexpected id error: %s", err))

	cases := []struct {
		desc string
		id   string
		err  error
	}{
		{
			desc: "retrieve an existing run",
			id:   run.ID,
			err:  nil,
		},
		{
			desc: "retrieve a non-existent run",
			id:   nonexistent,
			err:  dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		got, err := repo.RetrieveRunByID(ctx, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
		if tc.err == nil {
			assert.Equal(t, pyscripts.ScriptRunStatusFail, got.Status, fmt.Sprintf("%s: expected status to round-trip", tc.desc))
			assert.Equal(t, "boom", got.Error, fmt.Sprintf("%s: expected error to round-trip", tc.desc))
		}
	}
}

func TestRetrieveRunsByScript(t *testing.T) {
	cleanUp(t)
	repo := postgres.NewScriptRepository(dbutil.NewDatabase(db))
	ctx := context.Background()

	s := saveScriptFixture(t, repo)
	other := saveScriptFixture(t, repo)

	for i := 0; i < 3; i++ {
		_, err := repo.SaveRuns(ctx, newRun(t, s.ID, pyscripts.ScriptRunStatusSuccess, float64(i), nil))
		require.Nil(t, err, fmt.Sprintf("unexpected save error: %s", err))
	}
	_, err := repo.SaveRuns(ctx, newRun(t, s.ID, pyscripts.ScriptRunStatusFail, nil, nil))
	require.Nil(t, err, fmt.Sprintf("unexpected save error: %s", err))
	_, err = repo.SaveRuns(ctx, newRun(t, other.ID, pyscripts.ScriptRunStatusSuccess, nil, nil))
	require.Nil(t, err, fmt.Sprintf("unexpected save error: %s", err))

	cases := []struct {
		desc     string
		scriptID string
		pm       pyscripts.PageMetadata
		total    uint64
		size     int
	}{
		{
			desc:     "retrieve all runs of a script",
			scriptID: s.ID,
			pm:       pyscripts.PageMetadata{Limit: 10},
			total:    4,
			size:     4,
		},
		{
			desc:     "retrieve runs with a limit",
			scriptID: s.ID,
			pm:       pyscripts.PageMetadata{Limit: 2},
			total:    4,
			size:     2,
		},
		{
			desc:     "retrieve runs filtered by status",
			scriptID: s.ID,
			pm:       pyscripts.PageMetadata{Status: pyscripts.ScriptRunStatusFail, Limit: 10},
			total:    1,
			size:     1,
		},
		{
			desc:     "retrieve runs of another script",
			scriptID: other.ID,
			pm:       pyscripts.PageMetadata{Limit: 10},
			total:    1,
			size:     1,
		},
	}

	for _, tc := range cases {
		page, err := repo.RetrieveRunsByScript(ctx, tc.scriptID, tc.pm)
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))
		assert.Equal(t, tc.total, page.Total, fmt.Sprintf("%s: expected total %d got %d", tc.desc, tc.total, page.Total))
		assert.Len(t, page.Runs, tc.size, fmt.Sprintf("%s: expected %d runs got %d", tc.desc, tc.size, len(page.Runs)))
	}
}

func TestRemoveRuns(t *testing.T) {
	cleanUp(t)
	repo := postgres.NewScriptRepository(dbutil.NewDatabase(db))
	ctx := context.Background()

	s := saveScriptFixture(t, repo)

	run := newRun(t, s.ID, pyscripts.ScriptRunStatusSuccess, float64(1), nil)
	_, err := repo.SaveRuns(ctx, run)
	require.Nil(t, err, fmt.Sprintf("unexpected save error: %s", err))

	err = repo.RemoveRuns(ctx, run.ID)
	require.Nil(t, err, fmt.Sprintf("unexpected remove error: %s", err))

	_, err = repo.RetrieveRunByID(ctx, run.ID)
	assert.True(t, errors.Contains(err, dbutil.ErrNotFound), "expected run to be removed")
}
