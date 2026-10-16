// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package filestore_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MainfluxLabs/mainflux/filestore"
	fsmocks "github.com/MainfluxLabs/mainflux/filestore/mocks"
	"github.com/MainfluxLabs/mainflux/filestore/store"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	token          = "tok"
	groupID        = "group-1"
	thingID        = "thing-1"
	thingKey       = "thingkey-1"
	otherGID       = "group-2"
	binariesClass  = "binaries"
	documentsClass = "documents"
	binFormat      = "bin"
	pdfFormat      = "pdf"
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

	fi := filestore.FileInfo{Name: "a.pdf", Class: documentsClass, Format: pdfFormat}
	err := svc.SaveGroupFile(context.Background(), strings.NewReader("hello"), token, groupID, fi)
	assert.Nil(t, err, "save failed")

	// File written to store with checksum round-trip.
	got, statErr := os.Stat(filepath.Join(base, "groups", groupID, fi.Name))
	assert.Nil(t, statErr, "file missing on backend")
	assert.Equal(t, int64(5), got.Size())

	assert.Equal(t, 1, grRepo.Len())

	stored, err := grRepo.Retrieve(context.Background(), groupID, fi)
	require.Nil(t, err)
	assert.Equal(t, int64(5), stored.Size, "size not persisted")

	// Unauthorised user blocked before store.Put.
	err = svc.SaveGroupFile(context.Background(), strings.NewReader("x"), "nope", groupID, fi)
	assert.Error(t, err, "auth not enforced")
}

func TestSaveGroupFileCleanup(t *testing.T) {
	svc, grRepo, _, base := newSvc(t)

	grRepo.FailOn = "bad.pdf"
	fi := filestore.FileInfo{Name: "bad.pdf", Class: documentsClass, Format: pdfFormat}

	err := svc.SaveGroupFile(context.Background(), strings.NewReader("hello"), token, groupID, fi)
	assert.Error(t, err, "expected DB save failure")

	_, statErr := os.Stat(filepath.Join(base, "groups", groupID, "bad.pdf"))
	assert.True(t, os.IsNotExist(statErr), "orphaned object not cleaned")
	assert.Equal(t, 0, grRepo.Len())
}

func TestSaveGroupFileDuplicate(t *testing.T) {
	svc, grRepo, _, _ := newSvc(t)

	fi := filestore.FileInfo{Name: "a.pdf", Class: documentsClass, Format: pdfFormat}
	require.Nil(t, svc.SaveGroupFile(context.Background(), strings.NewReader("original"), token, groupID, fi))

	err := svc.SaveGroupFile(context.Background(), strings.NewReader("different"), token, groupID, fi)
	assert.True(t, errors.Contains(err, dbutil.ErrConflict), "expected conflict, got %v", err)

	assert.Equal(t, 1, grRepo.Len())

	rc, err := svc.ViewGroupFile(context.Background(), token, groupID, fi, nil)
	require.Nil(t, err)
	defer rc.Close()

	data, err := io.ReadAll(rc)
	require.Nil(t, err)
	assert.Equal(t, []byte("original"), data)
}

func TestRemoveGroupFile(t *testing.T) {
	svc, grRepo, _, base := newSvc(t)

	fi := filestore.FileInfo{Name: "a.pdf", Class: documentsClass, Format: pdfFormat}
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

	fi := filestore.FileInfo{Name: "a.pdf", Class: documentsClass, Format: pdfFormat}
	require.Nil(t, svc.SaveGroupFile(context.Background(), strings.NewReader("payload"), token, groupID, fi))

	rc, err := svc.ViewGroupFile(context.Background(), token, groupID, fi, nil)
	require.Nil(t, err)
	defer rc.Close()

	data, err := io.ReadAll(rc)
	require.Nil(t, err)
	assert.Equal(t, []byte("payload"), data)
}

func TestViewGroupFileChecksum(t *testing.T) {
	svc, _, _, base := newSvc(t)

	fi := filestore.FileInfo{Name: "a.pdf", Class: documentsClass, Format: pdfFormat}
	require.Nil(t, svc.SaveGroupFile(context.Background(), strings.NewReader("payload"), token, groupID, fi))

	// Corrupt bytes on disk; DB still has checksum of original.
	require.Nil(t, os.WriteFile(filepath.Join(base, "groups", groupID, fi.Name), []byte("tampered"), 0o644))

	rc, err := svc.ViewGroupFile(context.Background(), token, groupID, fi, nil)
	require.Nil(t, err)
	defer rc.Close()

	_, err = io.ReadAll(rc)
	assert.True(t, errors.Contains(err, store.ErrChecksumMismatch), "expected checksum mismatch, got %v", err)
}

func TestViewGroupFileMissing(t *testing.T) {
	svc, _, _, base := newSvc(t)

	fi := filestore.FileInfo{Name: "a.pdf", Class: documentsClass, Format: pdfFormat}
	require.Nil(t, svc.SaveGroupFile(context.Background(), strings.NewReader("payload"), token, groupID, fi))

	require.Nil(t, os.Remove(filepath.Join(base, "groups", groupID, fi.Name)))

	_, err := svc.ViewGroupFile(context.Background(), token, groupID, fi, nil)
	require.NotNil(t, err)
	assert.True(t, errors.Contains(err, dbutil.ErrNotFound), "expected not-found, got %v", err)
}

func TestViewGroupFileRange(t *testing.T) {
	svc, _, _, _ := newSvc(t)

	content := "hello mainflux"
	fi := filestore.FileInfo{Name: "a.bin", Class: binariesClass, Format: binFormat}
	require.Nil(t, svc.SaveGroupFile(context.Background(), strings.NewReader(content), token, groupID, fi))

	cases := []struct {
		desc      string
		byteRange *filestore.Range
		want      string
		partial   bool
		offset    int64
		err       error
	}{
		{
			desc:      "view whole file without range",
			byteRange: nil,
			want:      content,
			partial:   false,
			err:       nil,
		},
		{
			desc:      "view bounded range",
			byteRange: &filestore.Range{Start: 6, End: 9},
			want:      "main",
			partial:   true,
			offset:    6,
			err:       nil,
		},
		{
			desc:      "view open-ended range",
			byteRange: &filestore.Range{Start: 6, End: -1},
			want:      "mainflux",
			partial:   true,
			offset:    6,
			err:       nil,
		},
		{
			desc:      "view range with end past file size",
			byteRange: &filestore.Range{Start: 10, End: 100},
			want:      "flux",
			partial:   true,
			offset:    10,
			err:       nil,
		},
		{
			desc:      "view suffix range",
			byteRange: &filestore.Range{Start: -3, End: -1},
			want:      "lux",
			partial:   true,
			offset:    11,
			err:       nil,
		},
		{
			desc:      "view suffix range longer than file",
			byteRange: &filestore.Range{Start: -100, End: -1},
			want:      content,
			partial:   true,
			offset:    0,
			err:       nil,
		},
		{
			desc:      "view range starting past file size",
			byteRange: &filestore.Range{Start: 14, End: -1},
			err:       filestore.ErrRangeNotSatisfiable,
		},
	}

	for _, tc := range cases {
		file, err := svc.ViewGroupFile(context.Background(), token, groupID, fi, tc.byteRange)
		if tc.err != nil {
			assert.True(t, errors.Contains(err, tc.err), "%s: expected %v, got %v", tc.desc, tc.err, err)
			continue
		}
		require.Nil(t, err, "%s: unexpected error: %v", tc.desc, err)

		data, err := io.ReadAll(file)
		file.Close()
		require.Nil(t, err, "%s: read failed: %v", tc.desc, err)
		assert.Equal(t, tc.want, string(data), "%s: content mismatch", tc.desc)
		assert.Equal(t, tc.partial, file.Partial, "%s: partial mismatch", tc.desc)
		assert.Equal(t, int64(len(content)), file.Size, "%s: size mismatch", tc.desc)
		if tc.partial {
			assert.Equal(t, tc.offset, file.Offset, "%s: offset mismatch", tc.desc)
			assert.Equal(t, int64(len(tc.want)), file.Length, "%s: length mismatch", tc.desc)
		}
	}
}

func TestViewRangeNoChecksum(t *testing.T) {
	svc, _, _, base := newSvc(t)

	fi := filestore.FileInfo{Name: "a.bin", Class: binariesClass, Format: binFormat}
	require.Nil(t, svc.SaveGroupFile(context.Background(), strings.NewReader("payload"), token, groupID, fi))

	// Same size as the original, so only the checksum can tell them apart.
	require.Nil(t, os.WriteFile(filepath.Join(base, "groups", groupID, fi.Name), []byte("PAYLOAD"), 0o644))

	file, err := svc.ViewGroupFile(context.Background(), token, groupID, fi, &filestore.Range{Start: 3, End: -1})
	require.Nil(t, err)
	defer file.Close()

	data, err := io.ReadAll(file)
	require.Nil(t, err, "range read must not verify the whole-file checksum")
	assert.Equal(t, "LOAD", string(data))
}

func TestViewRangeNoSize(t *testing.T) {
	svc, grRepo, _, base := newSvc(t)

	// A file stored before sizes were recorded has size 0.
	fi := filestore.FileInfo{Name: "a.bin", Class: binariesClass, Format: binFormat}
	require.Nil(t, grRepo.Save(context.Background(), groupID, fi))
	dir := filepath.Join(base, "groups", groupID)
	require.Nil(t, os.MkdirAll(dir, 0o755))
	require.Nil(t, os.WriteFile(filepath.Join(dir, fi.Name), []byte("payload"), 0o644))

	file, err := svc.ViewGroupFile(context.Background(), token, groupID, fi, &filestore.Range{Start: 3, End: -1})
	require.Nil(t, err)
	defer file.Close()

	data, err := io.ReadAll(file)
	require.Nil(t, err)
	assert.Equal(t, "payload", string(data))
	assert.False(t, file.Partial, "range must be ignored when size is unknown")
}

func TestViewGroupFileByKeyRange(t *testing.T) {
	svc, _, _, _ := newSvc(t)

	fi := filestore.FileInfo{Name: "a.bin", Class: binariesClass, Format: binFormat}
	require.Nil(t, svc.SaveGroupFile(context.Background(), strings.NewReader("payload"), token, groupID, fi))

	file, err := svc.ViewGroupFileByKey(context.Background(), thingKey, fi, &filestore.Range{Start: 3, End: -1})
	require.Nil(t, err)
	defer file.Close()

	data, err := io.ReadAll(file)
	require.Nil(t, err)
	assert.Equal(t, "load", string(data))
	assert.True(t, file.Partial)
}

func TestRemoveAllFilesByGroup(t *testing.T) {
	svc, grRepo, thRepo, base := newSvc(t)

	// Seed a group file + two thing files in same group.
	require.Nil(t, svc.SaveGroupFile(context.Background(), strings.NewReader("g"), token, groupID,
		filestore.FileInfo{Name: "g.pdf", Class: documentsClass, Format: pdfFormat}))

	for _, id := range []string{"t-a", "t-b"} {
		require.Nil(t, thRepo.Save(context.Background(), id, groupID, filestore.FileInfo{Name: "x.bin", Class: binariesClass, Format: binFormat}))
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

func TestViewGroupFileByKeyACL(t *testing.T) {
	svc, _, _, _ := newSvc(t)

	fi := filestore.FileInfo{Name: "a.pdf", Class: documentsClass, Format: pdfFormat}
	require.Nil(t, svc.SaveGroupFile(context.Background(), strings.NewReader("payload"), token, groupID, fi))

	rc, err := svc.ViewGroupFileByKey(context.Background(), thingKey, fi, nil)
	require.Nil(t, err, "thing in group must access group file")
	rc.Close()
}

func TestSaveFile(t *testing.T) {
	svc, _, thRepo, base := newSvc(t)

	fi := filestore.FileInfo{Name: "x.bin", Class: binariesClass, Format: binFormat}
	err := svc.SaveFile(context.Background(), strings.NewReader("hello"), thingKey, fi)
	assert.Nil(t, err, "save failed")

	// Object written to store under things/<id>/<name>.
	got, statErr := os.Stat(filepath.Join(base, "things", thingID, fi.Name))
	assert.Nil(t, statErr, "file missing on backend")
	assert.Equal(t, int64(5), got.Size())

	// Checksum from store.Put persisted to the index row.
	stored, err := thRepo.Retrieve(context.Background(), thingID, fi)
	require.Nil(t, err)
	assert.NotEmpty(t, stored.Checksum, "checksum not persisted")
	assert.Equal(t, int64(5), stored.Size, "size not persisted")
}

func TestSaveFileDuplicate(t *testing.T) {
	svc, _, _, base := newSvc(t)

	fi := filestore.FileInfo{Name: "x.bin", Class: binariesClass, Format: binFormat}
	require.Nil(t, svc.SaveFile(context.Background(), strings.NewReader("original"), thingKey, fi))

	err := svc.SaveFile(context.Background(), strings.NewReader("different"), thingKey, fi)
	assert.True(t, errors.Contains(err, dbutil.ErrConflict), "expected conflict, got %v", err)

	data, statErr := os.ReadFile(filepath.Join(base, "things", thingID, fi.Name))
	require.Nil(t, statErr)
	assert.Equal(t, []byte("original"), data)
}

func TestViewFile(t *testing.T) {
	svc, _, _, _ := newSvc(t)

	fi := filestore.FileInfo{Name: "x.bin", Class: binariesClass, Format: binFormat}
	require.Nil(t, svc.SaveFile(context.Background(), strings.NewReader("payload"), thingKey, fi))

	rc, err := svc.ViewFile(context.Background(), thingKey, fi, nil)
	require.Nil(t, err)
	defer rc.Close()

	data, err := io.ReadAll(rc)
	require.Nil(t, err)
	assert.Equal(t, []byte("payload"), data)
}

func TestViewFileRange(t *testing.T) {
	svc, _, _, _ := newSvc(t)

	fi := filestore.FileInfo{Name: "x.bin", Class: binariesClass, Format: binFormat}
	require.Nil(t, svc.SaveFile(context.Background(), strings.NewReader("hello"), thingKey, fi))

	file, err := svc.ViewFile(context.Background(), thingKey, fi, &filestore.Range{Start: 1, End: 2})
	require.Nil(t, err)
	defer file.Close()

	data, err := io.ReadAll(file)
	require.Nil(t, err)
	assert.Equal(t, "el", string(data))
	assert.True(t, file.Partial)
}

func TestViewFileChecksum(t *testing.T) {
	svc, _, _, base := newSvc(t)

	fi := filestore.FileInfo{Name: "x.bin", Class: binariesClass, Format: binFormat}
	require.Nil(t, svc.SaveFile(context.Background(), strings.NewReader("payload"), thingKey, fi))

	// Corrupt bytes on disk; index still has checksum of original.
	require.Nil(t, os.WriteFile(filepath.Join(base, "things", thingID, fi.Name), []byte("tampered"), 0o644))

	rc, err := svc.ViewFile(context.Background(), thingKey, fi, nil)
	require.Nil(t, err)
	defer rc.Close()

	_, err = io.ReadAll(rc)
	assert.True(t, errors.Contains(err, store.ErrChecksumMismatch), "expected checksum mismatch, got %v", err)
}

func TestRemoveFile(t *testing.T) {
	svc, _, _, base := newSvc(t)

	fi := filestore.FileInfo{Name: "x.bin", Class: binariesClass, Format: binFormat}
	require.Nil(t, svc.SaveFile(context.Background(), strings.NewReader("hello"), thingKey, fi))

	err := svc.RemoveFile(context.Background(), thingKey, fi)
	assert.Nil(t, err, "remove failed")

	_, statErr := os.Stat(filepath.Join(base, "things", thingID, fi.Name))
	assert.True(t, os.IsNotExist(statErr), "object still present after remove")
}

func TestRemoveFiles(t *testing.T) {
	svc, _, _, base := newSvc(t)

	for _, n := range []string{"a.bin", "b.bin"} {
		require.Nil(t, svc.SaveFile(context.Background(), strings.NewReader("x"), thingKey,
			filestore.FileInfo{Name: n, Class: binariesClass, Format: binFormat}))
	}

	err := svc.RemoveFiles(context.Background(), thingID)
	assert.Nil(t, err, "remove all failed")

	_, statErr := os.Stat(filepath.Join(base, "things", thingID))
	assert.True(t, os.IsNotExist(statErr), "thing prefix not cleared")
}

func TestChecksumMismatchPropagates(t *testing.T) {
	svc, _, _, base := newSvc(t)

	fi := filestore.FileInfo{Name: "a.pdf", Class: documentsClass, Format: pdfFormat}
	require.Nil(t, svc.SaveGroupFile(context.Background(), bytes.NewReader([]byte("payload")), token, groupID, fi))

	path := filepath.Join(base, "groups", groupID, fi.Name)
	require.Nil(t, os.WriteFile(path, []byte("tampered-different-length"), 0o644))

	rc, err := svc.ViewGroupFile(context.Background(), token, groupID, fi, nil)
	require.Nil(t, err)
	defer rc.Close()

	_, err = io.ReadAll(rc)
	assert.True(t, errors.Contains(err, store.ErrChecksumMismatch))
}
