// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package pyscripts_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	pkgmocks "github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/pyscripts"
	psmocks "github.com/MainfluxLabs/mainflux/pyscripts/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	email         = "admin@example.com"
	token         = email
	wrongToken    = "wrong-token"
	groupID       = "574106f7-030e-4881-8ab0-151195c29f94"
	wrongID       = "wrong-id"
	validSource   = "1 + 1"
	badSource     = "def (:"
	maxScriptSize = 1024
)

func newService() (pyscripts.Service, *psmocks.Runner) {
	things := pkgmocks.NewThingsServiceClient(
		nil,
		map[string]domain.Thing{token: {ID: "thing", GroupID: groupID}},
		map[string]domain.Group{token: {ID: groupID}},
	)
	repo := psmocks.NewScriptRepository()
	run := psmocks.NewRunner()
	run.Invalid[badSource] = "invalid syntax"
	idp := uuid.NewMock()

	return pyscripts.New(repo, run, things, idp, logger.NewMock(), maxScriptSize), run
}

func script(name, source string) pyscripts.Script {
	return pyscripts.Script{Name: name, Source: source}
}

func TestCreateScripts(t *testing.T) {
	svc, _ := newService()

	cases := []struct {
		desc    string
		token   string
		groupID string
		scripts []pyscripts.Script
		num     int
		err     error
	}{
		{
			desc:    "create scripts with valid token",
			token:   token,
			groupID: groupID,
			scripts: []pyscripts.Script{script("a", validSource), script("b", validSource)},
			num:     2,
			err:     nil,
		},
		{
			desc:    "create scripts with empty token",
			token:   "",
			groupID: groupID,
			scripts: []pyscripts.Script{script("c", validSource)},
			err:     errors.ErrAuthentication,
		},
		{
			desc:    "create scripts with wrong token",
			token:   wrongToken,
			groupID: groupID,
			scripts: []pyscripts.Script{script("c", validSource)},
			err:     errors.ErrAuthentication,
		},
		{
			desc:    "create scripts for group the user cannot access",
			token:   token,
			groupID: wrongID,
			scripts: []pyscripts.Script{script("c", validSource)},
			err:     errors.ErrAuthorization,
		},
		{
			desc:    "create scripts with source over the size limit",
			token:   token,
			groupID: groupID,
			scripts: []pyscripts.Script{script("d", strings.Repeat("a", maxScriptSize+1))},
			err:     pyscripts.ErrScriptSize,
		},
		{
			desc:    "create scripts with source that does not compile",
			token:   token,
			groupID: groupID,
			scripts: []pyscripts.Script{script("e", badSource)},
			err:     pyscripts.ErrScriptSyntax,
		},
	}

	for _, tc := range cases {
		saved, err := svc.CreateScripts(context.Background(), tc.token, tc.groupID, tc.scripts...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
		if tc.err == nil {
			assert.Len(t, saved, tc.num, fmt.Sprintf("%s: expected %d scripts", tc.desc, tc.num))
			for _, s := range saved {
				assert.NotEmpty(t, s.ID, fmt.Sprintf("%s: expected an assigned id", tc.desc))
				assert.Equal(t, tc.groupID, s.GroupID, fmt.Sprintf("%s: expected group id to be set", tc.desc))
				assert.Equal(t, pyscripts.Checksum(s.Source), s.Checksum, fmt.Sprintf("%s: expected checksum of the source", tc.desc))
			}
		}
	}
}

func TestListScriptsByGroup(t *testing.T) {
	svc, _ := newService()

	_, err := svc.CreateScripts(context.Background(), token, groupID, script("a", validSource), script("b", validSource), script("c", validSource))
	require.Nil(t, err, fmt.Sprintf("unexpected create error: %s", err))

	cases := []struct {
		desc    string
		token   string
		groupID string
		pm      pyscripts.PageMetadata
		total   uint64
		size    int
		err     error
	}{
		{
			desc:    "list scripts with valid token",
			token:   token,
			groupID: groupID,
			pm:      pyscripts.PageMetadata{Limit: 10},
			total:   3,
			size:    3,
			err:     nil,
		},
		{
			desc:    "list scripts with a limit",
			token:   token,
			groupID: groupID,
			pm:      pyscripts.PageMetadata{Limit: 2},
			total:   3,
			size:    2,
			err:     nil,
		},
		{
			desc:    "list scripts with an offset",
			token:   token,
			groupID: groupID,
			pm:      pyscripts.PageMetadata{Offset: 2, Limit: 10},
			total:   3,
			size:    1,
			err:     nil,
		},
		{
			desc:    "list scripts with empty token",
			token:   "",
			groupID: groupID,
			pm:      pyscripts.PageMetadata{Limit: 10},
			err:     errors.ErrAuthentication,
		},
		{
			desc:    "list scripts for group the user cannot access",
			token:   token,
			groupID: wrongID,
			pm:      pyscripts.PageMetadata{Limit: 10},
			err:     errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		page, err := svc.ListScriptsByGroup(context.Background(), tc.token, tc.groupID, tc.pm)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
		if tc.err == nil {
			assert.Equal(t, tc.total, page.Total, fmt.Sprintf("%s: expected total %d got %d", tc.desc, tc.total, page.Total))
			assert.Len(t, page.Scripts, tc.size, fmt.Sprintf("%s: expected %d scripts got %d", tc.desc, tc.size, len(page.Scripts)))
		}
	}
}

func TestViewScript(t *testing.T) {
	svc, _ := newService()

	saved, err := svc.CreateScripts(context.Background(), token, groupID, script("a", validSource))
	require.Nil(t, err, fmt.Sprintf("unexpected create error: %s", err))
	id := saved[0].ID

	cases := []struct {
		desc  string
		token string
		id    string
		err   error
	}{
		{
			desc:  "view script with valid token",
			token: token,
			id:    id,
			err:   nil,
		},
		{
			desc:  "view non-existent script",
			token: token,
			id:    wrongID,
			err:   dbutil.ErrNotFound,
		},
		{
			desc:  "view script with empty token",
			token: "",
			id:    id,
			err:   errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		s, err := svc.ViewScript(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
		if tc.err == nil {
			assert.Equal(t, id, s.ID, fmt.Sprintf("%s: expected script %s got %s", tc.desc, id, s.ID))
			assert.Equal(t, validSource, s.Source, fmt.Sprintf("%s: expected source to be returned", tc.desc))
		}
	}
}

func TestUpdateScript(t *testing.T) {
	svc, _ := newService()

	saved, err := svc.CreateScripts(context.Background(), token, groupID, script("a", validSource))
	require.Nil(t, err, fmt.Sprintf("unexpected create error: %s", err))
	id := saved[0].ID

	cases := []struct {
		desc   string
		token  string
		script pyscripts.Script
		err    error
	}{
		{
			desc:   "update script with valid token",
			token:  token,
			script: pyscripts.Script{ID: id, Name: "renamed", Source: "2 + 2"},
			err:    nil,
		},
		{
			desc:   "update non-existent script",
			token:  token,
			script: pyscripts.Script{ID: wrongID, Name: "x", Source: validSource},
			err:    dbutil.ErrNotFound,
		},
		{
			desc:   "update script with empty token",
			token:  "",
			script: pyscripts.Script{ID: id, Name: "x", Source: validSource},
			err:    errors.ErrAuthentication,
		},
		{
			desc:   "update script with source that does not compile",
			token:  token,
			script: pyscripts.Script{ID: id, Name: "x", Source: badSource},
			err:    pyscripts.ErrScriptSyntax,
		},
	}

	for _, tc := range cases {
		err := svc.UpdateScript(context.Background(), tc.token, tc.script)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
		if tc.err == nil {
			s, err := svc.ViewScript(context.Background(), token, tc.script.ID)
			require.Nil(t, err, fmt.Sprintf("%s: unexpected view error: %s", tc.desc, err))
			assert.Equal(t, tc.script.Source, s.Source, fmt.Sprintf("%s: expected updated source", tc.desc))
			assert.Equal(t, pyscripts.Checksum(tc.script.Source), s.Checksum, fmt.Sprintf("%s: expected recomputed checksum", tc.desc))
		}
	}
}

func TestRemoveScripts(t *testing.T) {
	svc, _ := newService()

	saved, err := svc.CreateScripts(context.Background(), token, groupID, script("a", validSource), script("b", validSource))
	require.Nil(t, err, fmt.Sprintf("unexpected create error: %s", err))

	cases := []struct {
		desc  string
		token string
		ids   []string
		err   error
	}{
		{
			desc:  "remove script with empty token",
			token: "",
			ids:   []string{saved[0].ID},
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "remove non-existent script",
			token: token,
			ids:   []string{wrongID},
			err:   dbutil.ErrNotFound,
		},
		{
			desc:  "remove script with valid token",
			token: token,
			ids:   []string{saved[0].ID},
			err:   nil,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveScripts(context.Background(), tc.token, tc.ids...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
		if tc.err == nil {
			_, err := svc.ViewScript(context.Background(), token, tc.ids[0])
			assert.True(t, errors.Contains(err, dbutil.ErrNotFound), fmt.Sprintf("%s: expected script to be gone", tc.desc))
		}
	}
}

func TestRemoveScriptsByGroup(t *testing.T) {
	svc, _ := newService()

	saved, err := svc.CreateScripts(context.Background(), token, groupID, script("a", validSource), script("b", validSource))
	require.Nil(t, err, fmt.Sprintf("unexpected create error: %s", err))

	err = svc.RemoveScriptsByGroup(context.Background(), groupID)
	require.Nil(t, err, fmt.Sprintf("unexpected remove-by-group error: %s", err))

	for _, s := range saved {
		_, err := svc.ViewScript(context.Background(), token, s.ID)
		assert.True(t, errors.Contains(err, dbutil.ErrNotFound), "expected every script in the group to be removed")
	}
}

func TestRunScript(t *testing.T) {
	svc, run := newService()

	saved, err := svc.CreateScripts(context.Background(), token, groupID, script("a", validSource))
	require.Nil(t, err, fmt.Sprintf("unexpected create error: %s", err))
	id := saved[0].ID

	cases := []struct {
		desc     string
		token    string
		id       string
		result   pyscripts.Result
		runErr   error
		status   string
		runSaved bool
		err      error
	}{
		{
			desc:     "run script that succeeds",
			token:    token,
			id:       id,
			result:   pyscripts.Result{Value: float64(2), Logs: []string{"hello"}},
			status:   pyscripts.ScriptRunStatusSuccess,
			runSaved: true,
			err:      nil,
		},
		{
			desc:     "run script whose code reports an error",
			token:    token,
			id:       id,
			result:   pyscripts.Result{Error: "boom"},
			status:   pyscripts.ScriptRunStatusFail,
			runSaved: true,
			err:      nil,
		},
		{
			desc:   "run script when the runner fails",
			token:  token,
			id:     id,
			runErr: pyscripts.ErrRunnerFailed,
			err:    pyscripts.ErrRunnerFailed,
		},
		{
			desc:  "run non-existent script",
			token: token,
			id:    wrongID,
			err:   dbutil.ErrNotFound,
		},
		{
			desc:  "run script with empty token",
			token: "",
			id:    id,
			err:   errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		run.Result = tc.result
		run.RunErr = tc.runErr

		r, err := svc.RunScript(context.Background(), tc.token, tc.id, pyscripts.Input{Payload: map[string]any{"x": 1}})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
		if tc.err == nil {
			assert.Equal(t, tc.status, r.Status, fmt.Sprintf("%s: expected status %s got %s", tc.desc, tc.status, r.Status))
			assert.Equal(t, id, r.ScriptID, fmt.Sprintf("%s: expected run tied to script", tc.desc))
			assert.NotEmpty(t, r.ID, fmt.Sprintf("%s: expected an assigned run id", tc.desc))

			stored, err := svc.ListRunsByScript(context.Background(), token, id, pyscripts.PageMetadata{Limit: 100})
			require.Nil(t, err, fmt.Sprintf("%s: unexpected list error: %s", tc.desc, err))
			found := false
			for _, sr := range stored.Runs {
				if sr.ID == r.ID {
					found = true
				}
			}
			assert.Equal(t, tc.runSaved, found, fmt.Sprintf("%s: expected run persisted=%v", tc.desc, tc.runSaved))
		}
	}
}

func TestListRunsByScript(t *testing.T) {
	svc, run := newService()

	saved, err := svc.CreateScripts(context.Background(), token, groupID, script("a", validSource))
	require.Nil(t, err, fmt.Sprintf("unexpected create error: %s", err))
	id := saved[0].ID

	run.Result = pyscripts.Result{Value: float64(1)}
	for i := 0; i < 3; i++ {
		_, err := svc.RunScript(context.Background(), token, id, pyscripts.Input{})
		require.Nil(t, err, fmt.Sprintf("unexpected run error: %s", err))
	}

	cases := []struct {
		desc  string
		token string
		id    string
		total uint64
		err   error
	}{
		{
			desc:  "list runs with valid token",
			token: token,
			id:    id,
			total: 3,
			err:   nil,
		},
		{
			desc:  "list runs for non-existent script",
			token: token,
			id:    wrongID,
			err:   dbutil.ErrNotFound,
		},
		{
			desc:  "list runs with empty token",
			token: "",
			id:    id,
			err:   errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		page, err := svc.ListRunsByScript(context.Background(), tc.token, tc.id, pyscripts.PageMetadata{Limit: 10})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
		if tc.err == nil {
			assert.Equal(t, tc.total, page.Total, fmt.Sprintf("%s: expected total %d got %d", tc.desc, tc.total, page.Total))
		}
	}
}

func TestRemoveRuns(t *testing.T) {
	svc, run := newService()

	saved, err := svc.CreateScripts(context.Background(), token, groupID, script("a", validSource))
	require.Nil(t, err, fmt.Sprintf("unexpected create error: %s", err))
	id := saved[0].ID

	run.Result = pyscripts.Result{Value: float64(1)}
	r, err := svc.RunScript(context.Background(), token, id, pyscripts.Input{})
	require.Nil(t, err, fmt.Sprintf("unexpected run error: %s", err))

	cases := []struct {
		desc  string
		token string
		ids   []string
		err   error
	}{
		{
			desc:  "remove run with empty token",
			token: "",
			ids:   []string{r.ID},
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "remove non-existent run",
			token: token,
			ids:   []string{wrongID},
			err:   dbutil.ErrNotFound,
		},
		{
			desc:  "remove run with valid token",
			token: token,
			ids:   []string{r.ID},
			err:   nil,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveRuns(context.Background(), tc.token, tc.ids...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}

func TestRunScriptRateLimit(t *testing.T) {
	const (
		token1 = "tenant-one"
		token2 = "tenant-two"
		group1 = "11111111-1111-1111-1111-111111111111"
		group2 = "22222222-2222-2222-2222-222222222222"
	)

	things := pkgmocks.NewThingsServiceClient(
		nil,
		map[string]domain.Thing{},
		map[string]domain.Group{
			token1: {ID: group1},
			token2: {ID: group2},
		},
	)
	svc := pyscripts.New(psmocks.NewScriptRepository(), psmocks.NewRunner(), things, uuid.NewMock(), logger.NewMock(), maxScriptSize)

	s1, err := svc.CreateScripts(context.Background(), token1, group1, script("a", validSource))
	require.Nil(t, err, fmt.Sprintf("unexpected create error: %s", err))
	s2, err := svc.CreateScripts(context.Background(), token2, group2, script("b", validSource))
	require.Nil(t, err, fmt.Sprintf("unexpected create error: %s", err))

	var allowed, limited int
	for i := 0; i < 50; i++ {
		_, err := svc.RunScript(context.Background(), token1, s1[0].ID, pyscripts.Input{})
		switch {
		case err == nil:
			allowed++
		case errors.Contains(err, pyscripts.ErrRateLimited):
			limited++
		default:
			t.Fatalf("unexpected error on run %d: %s", i, err)
		}
	}

	assert.Greater(t, allowed, 0, "expected the initial burst to be allowed")
	assert.Greater(t, limited, 0, "expected a looping group to be rate limited")

	_, err = svc.RunScript(context.Background(), token2, s2[0].ID, pyscripts.Input{})
	assert.Nil(t, err, fmt.Sprintf("expected a different group to be unaffected by another's limit: %s", err))
}
