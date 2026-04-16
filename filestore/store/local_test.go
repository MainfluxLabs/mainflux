// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package store_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MainfluxLabs/mainflux/filestore/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testKey     = "groups/g1/file.txt"
	testContent = "hello mainflux"
	wrongSum    = "deadbeef"
)

func sum(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func newLocal(t *testing.T) (store.FileStore, string) {
	t.Helper()
	base := t.TempDir()
	return store.NewLocal(base), base
}

func TestLocalPut(t *testing.T) {
	s, base := newLocal(t)

	cases := []struct {
		desc    string
		key     string
		content []byte
		err     error
	}{
		{
			desc:    "put single file",
			key:     testKey,
			content: []byte(testContent),
			err:     nil,
		},
		{
			desc:    "put file with nested path",
			key:     "groups/g2/sub/dir/file.bin",
			content: []byte("nested"),
			err:     nil,
		},
		{
			desc:    "put empty file",
			key:     "groups/g3/empty.txt",
			content: []byte{},
			err:     nil,
		},
		{
			desc:    "put large file",
			key:     "groups/g4/big.bin",
			content: bytes.Repeat([]byte("x"), 1<<20),
			err:     nil,
		},
	}

	for _, tc := range cases {
		checksum, err := s.Put(context.Background(), tc.key, bytes.NewReader(tc.content))
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))
		assert.Equal(t, sum(tc.content), checksum, fmt.Sprintf("%s: checksum mismatch", tc.desc))

		data, readErr := os.ReadFile(filepath.Join(base, tc.key))
		require.Nil(t, readErr, fmt.Sprintf("%s: read back failed: %s", tc.desc, readErr))
		assert.Equal(t, tc.content, data, fmt.Sprintf("%s: content mismatch", tc.desc))
	}
}

func TestLocalGet(t *testing.T) {
	s, _ := newLocal(t)
	content := []byte(testContent)
	expected, err := s.Put(context.Background(), testKey, bytes.NewReader(content))
	require.Nil(t, err, fmt.Sprintf("put failed: %s", err))

	cases := []struct {
		desc     string
		key      string
		checksum string
		want     []byte
		err      error
	}{
		{
			desc:     "get with valid checksum",
			key:      testKey,
			checksum: expected,
			want:     content,
			err:      nil,
		},
		{
			desc:     "get without checksum",
			key:      testKey,
			checksum: "",
			want:     content,
			err:      nil,
		},
		{
			desc:     "get missing key",
			key:      "groups/missing/x.txt",
			checksum: "",
			want:     nil,
			err:      os.ErrNotExist,
		},
	}

	for _, tc := range cases {
		rc, err := s.Get(context.Background(), tc.key, tc.checksum)
		if tc.err != nil {
			assert.True(t, errors.Is(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
			continue
		}
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))
		data, readErr := io.ReadAll(rc)
		require.Nil(t, readErr, fmt.Sprintf("%s: read failed: %s", tc.desc, readErr))
		rc.Close()
		assert.Equal(t, tc.want, data, fmt.Sprintf("%s: content mismatch", tc.desc))
	}
}

func TestLocalGetChecksumMismatch(t *testing.T) {
	s, _ := newLocal(t)
	_, err := s.Put(context.Background(), testKey, bytes.NewReader([]byte(testContent)))
	require.Nil(t, err, fmt.Sprintf("put failed: %s", err))

	rc, err := s.Get(context.Background(), testKey, wrongSum)
	require.Nil(t, err, fmt.Sprintf("get open failed: %s", err))
	defer rc.Close()

	_, err = io.ReadAll(rc)
	assert.True(t, errors.Is(err, store.ErrChecksumMismatch), fmt.Sprintf("expected ErrChecksumMismatch got %s", err))
}

func TestLocalDelete(t *testing.T) {
	s, base := newLocal(t)
	_, err := s.Put(context.Background(), testKey, bytes.NewReader([]byte(testContent)))
	require.Nil(t, err, fmt.Sprintf("put failed: %s", err))

	cases := []struct {
		desc string
		key  string
		err  error
	}{
		{
			desc: "delete existing key",
			key:  testKey,
			err:  nil,
		},
		{
			desc: "delete missing key is no-op",
			key:  "groups/missing/x.txt",
			err:  nil,
		},
	}

	for _, tc := range cases {
		err := s.Delete(context.Background(), tc.key)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))
	}

	_, err = os.Stat(filepath.Join(base, testKey))
	assert.True(t, os.IsNotExist(err), "file should be gone")
}

func TestLocalDeleteAll(t *testing.T) {
	s, base := newLocal(t)
	keys := []string{}
	for i := 0; i < 50; i++ {
		k := fmt.Sprintf("groups/g/f%d.bin", i)
		_, err := s.Put(context.Background(), k, strings.NewReader("x"))
		require.Nil(t, err, fmt.Sprintf("put %d failed: %s", i, err))
		keys = append(keys, k)
	}

	cases := []struct {
		desc string
		keys []string
		err  error
	}{
		{
			desc: "delete empty slice",
			keys: []string{},
			err:  nil,
		},
		{
			desc: "delete many keys in parallel",
			keys: keys,
			err:  nil,
		},
		{
			desc: "delete mix of existing and missing",
			keys: []string{"groups/x/a", "groups/x/b"},
			err:  nil,
		},
	}

	for _, tc := range cases {
		err := s.DeleteAll(context.Background(), tc.keys)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))
	}

	for _, k := range keys {
		_, err := os.Stat(filepath.Join(base, k))
		assert.True(t, os.IsNotExist(err), fmt.Sprintf("%s should be gone", k))
	}
}

func TestLocalDeletePrefix(t *testing.T) {
	s, base := newLocal(t)
	for _, k := range []string{
		"groups/g1/a.txt",
		"groups/g1/sub/b.txt",
		"groups/g2/c.txt",
	} {
		_, err := s.Put(context.Background(), k, strings.NewReader("x"))
		require.Nil(t, err, fmt.Sprintf("put %s failed: %s", k, err))
	}

	cases := []struct {
		desc    string
		prefix  string
		remains []string
		gone    []string
		err     error
	}{
		{
			desc:    "delete prefix removes subtree",
			prefix:  "groups/g1",
			remains: []string{"groups/g2/c.txt"},
			gone:    []string{"groups/g1/a.txt", "groups/g1/sub/b.txt"},
			err:     nil,
		},
		{
			desc:    "delete missing prefix is no-op",
			prefix:  "groups/missing",
			remains: []string{"groups/g2/c.txt"},
			err:     nil,
		},
	}

	for _, tc := range cases {
		err := s.DeletePrefix(context.Background(), tc.prefix)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))

		for _, k := range tc.gone {
			_, statErr := os.Stat(filepath.Join(base, k))
			assert.True(t, os.IsNotExist(statErr), fmt.Sprintf("%s: %s should be gone", tc.desc, k))
		}
		for _, k := range tc.remains {
			_, statErr := os.Stat(filepath.Join(base, k))
			assert.Nil(t, statErr, fmt.Sprintf("%s: %s should remain", tc.desc, k))
		}
	}
}
