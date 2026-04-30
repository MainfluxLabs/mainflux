// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package filestore_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MainfluxLabs/mainflux/filestore"
	fsmocks "github.com/MainfluxLabs/mainflux/filestore/mocks"
	"github.com/MainfluxLabs/mainflux/filestore/store"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	token     = "tok"
	groupID   = "group-1"
	thingID   = "thing-1"
	thingKey  = "thingkey-1"
	otherGID  = "group-2"
	otherThID = "thing-2"
)

func newSvc(t *testing.T) (filestore.Service, *fsmocks.GroupsRepository, *fsmocks.ThingsRepository, string) {
	t.Helper()

	base := t.TempDir()
	fs := store.NewLocal(base)

	grRepo := fsmocks.NewGroupsRepository()
	thRepo := fsmocks.NewThingsRepository()

	things := map[string]domain.Thing{
		thingKey: {ID: thingID, GroupID: groupID, Key: thingKey},
		thingID:  {ID: thingID, GroupID: groupID, Key: thingKey},
		token:    {ID: thingID, GroupID: groupID},
	}
	groups := map[string]domain.Group{
		token:    {ID: groupID},
		otherGID: {ID: otherGID},
	}
	tc := mocks.NewThingsServiceClient(nil, things, groups)

	log, err := logger.New(io.Discard, "error")
	require.Nil(t, err)

	svc := filestore.New(tc, thRepo, grRepo, fs, log)
	return svc, grRepo, thRepo, base
}

func TestSaveGroupFile(t *testing.T) {
	svc, grRepo, _, base := newSvc(t)

	fi := filestore.FileInfo{Name: "a.pdf", Class: "documents", Format: "pdf"}
	err := svc.SaveGroupFile(context.Background(), strings.NewReader("hello"), token, groupID, fi)
	assert.Nil(t, err, "save failed")

	// File written to store with checksum round-trip.
	got, statErr := os.Stat(filepath.Join(base, "groups", groupID, fi.Name))
	assert.Nil(t, statErr, "file missing on backend")
	assert.Equal(t, int64(5), got.Size())

	assert.Equal(t, 1, grRepo.Len())

	// Unauthorised user blocked before store.Put.
	err = svc.SaveGroupFile(context.Background(), strings.NewReader("x"), "nope", groupID, fi)
	assert.Error(t, err, "auth not enforced")
}

func TestSaveGroupFile_OrphanCleanup(t *testing.T) {
	svc, grRepo, _, base := newSvc(t)

	grRepo.FailOn = "bad.pdf"
	fi := filestore.FileInfo{Name: "bad.pdf", Class: "documents", Format: "pdf"}

	err := svc.SaveGroupFile(context.Background(), strings.NewReader("hello"), token, groupID, fi)
	assert.Error(t, err, "expected DB save failure")

	_, statErr := os.Stat(filepath.Join(base, "groups", groupID, "bad.pdf"))
	assert.True(t, os.IsNotExist(statErr), "orphaned object not cleaned")
	assert.Equal(t, 0, grRepo.Len())
}

func TestRemoveGroupFile(t *testing.T) {
	svc, grRepo, _, base := newSvc(t)

	fi := filestore.FileInfo{Name: "a.pdf", Class: "documents", Format: "pdf"}
	require.Nil(t, svc.SaveGroupFile(context.Background(), strings.NewReader("hello"), token, groupID, fi))

	err := svc.RemoveGroupFile(context.Background(), token, groupID, fi)
	assert.Nil(t, err, "remove failed")

	_, statErr := os.Stat(filepath.Join(base, "groups", groupID, fi.Name))
	assert.True(t, os.IsNotExist(statErr), "object still present after remove")
	assert.Equal(t, 0, grRepo.Len())

	// Second remove reports ErrNotFound surfaced from repo.
	err = svc.RemoveGroupFile(context.Background(), token, groupID, fi)
	assert.Error(t, err, "expected not-found on double remove")
}

func TestViewGroupFile(t *testing.T) {
	svc, _, _, _ := newSvc(t)

	fi := filestore.FileInfo{Name: "a.pdf", Class: "documents", Format: "pdf"}
	require.Nil(t, svc.SaveGroupFile(context.Background(), strings.NewReader("payload"), token, groupID, fi))

	rc, err := svc.ViewGroupFile(context.Background(), token, groupID, fi)
	require.Nil(t, err)
	defer rc.Close()

	data, err := io.ReadAll(rc)
	require.Nil(t, err)
	assert.Equal(t, []byte("payload"), data)
}

func TestViewGroupFile_ChecksumMismatch(t *testing.T) {
	svc, grRepo, _, base := newSvc(t)

	fi := filestore.FileInfo{Name: "a.pdf", Class: "documents", Format: "pdf"}
	require.Nil(t, svc.SaveGroupFile(context.Background(), strings.NewReader("payload"), token, groupID, fi))

	// Corrupt bytes on disk; DB still has checksum of original.
	require.Nil(t, os.WriteFile(filepath.Join(base, "groups", groupID, fi.Name), []byte("tampered"), 0o644))

	rc, err := svc.ViewGroupFile(context.Background(), token, groupID, fi)
	require.Nil(t, err)
	defer rc.Close()

	_, err = io.ReadAll(rc)
	assert.True(t, errors.Is(err, store.ErrChecksumMismatch), "expected checksum mismatch, got %v", err)
	_ = grRepo
}

func TestRemoveAllFilesByGroup(t *testing.T) {
	svc, grRepo, thRepo, base := newSvc(t)

	// Seed a group file + two thing files in same group.
	require.Nil(t, svc.SaveGroupFile(context.Background(), strings.NewReader("g"), token, groupID,
		filestore.FileInfo{Name: "g.pdf", Class: "documents", Format: "pdf"}))

	for _, id := range []string{"t-a", "t-b"} {
		require.Nil(t, thRepo.Save(context.Background(), id, groupID, filestore.FileInfo{Name: "x.bin", Class: "binaries", Format: "bin"}))
		require.Nil(t, os.MkdirAll(filepath.Join(base, "things", id), 0o755))
		require.Nil(t, os.WriteFile(filepath.Join(base, "things", id, "x.bin"), []byte("payload"), 0o644))
	}

	err := svc.RemoveAllFilesByGroup(context.Background(), groupID)
	assert.Nil(t, err, "cascade failed")

	_, statErr := os.Stat(filepath.Join(base, "groups", groupID))
	assert.True(t, os.IsNotExist(statErr), "group prefix not cleared")
	for _, id := range []string{"t-a", "t-b"} {
		_, statErr := os.Stat(filepath.Join(base, "things", id))
		assert.True(t, os.IsNotExist(statErr), "thing %s prefix not cleared", id)
	}
	assert.Equal(t, 0, grRepo.Len())
}

func TestViewGroupFileByKey_ACL(t *testing.T) {
	svc, _, _, _ := newSvc(t)

	fi := filestore.FileInfo{Name: "a.pdf", Class: "documents", Format: "pdf"}
	require.Nil(t, svc.SaveGroupFile(context.Background(), strings.NewReader("payload"), token, groupID, fi))

	// Thing in same group: allowed.
	rc, err := svc.ViewGroupFileByKey(context.Background(), thingKey, fi)
	require.Nil(t, err, "thing in group must access group file")
	rc.Close()
}

// Smoke assertion: ErrChecksumMismatch actually surfaces through the service.
func TestChecksumMismatchPropagates(t *testing.T) {
	svc, _, _, base := newSvc(t)

	fi := filestore.FileInfo{Name: "a.pdf", Class: "documents", Format: "pdf"}
	require.Nil(t, svc.SaveGroupFile(context.Background(), bytes.NewReader([]byte("payload")), token, groupID, fi))

	path := filepath.Join(base, "groups", groupID, fi.Name)
	require.Nil(t, os.WriteFile(path, []byte("tampered-different-length"), 0o644))

	rc, err := svc.ViewGroupFile(context.Background(), token, groupID, fi)
	require.Nil(t, err)
	defer rc.Close()

	_, err = io.ReadAll(rc)
	assert.True(t, errors.Is(err, store.ErrChecksumMismatch))
}
