// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package store_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/filestore/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	swPrefix  = "filestore"
	swTimeout = 5 * time.Second
)

type fakeFiler struct {
	mu    sync.Mutex
	files map[string][]byte
}

func newFakeFiler() *fakeFiler {
	return &fakeFiler{files: map[string][]byte{}}
}

func (f *fakeFiler) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()

		key := strings.TrimLeft(r.URL.Path, "/")
		switch r.Method {
		case http.MethodPut:
			b, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			f.files[key] = b
			w.WriteHeader(http.StatusCreated)
		case http.MethodGet:
			b, ok := f.files[key]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Write(b)
		case http.MethodDelete:
			recursive := r.URL.Query().Get("recursive") == "true"
			key = strings.TrimRight(key, "/")
			if recursive {
				for k := range f.files {
					if k == key || strings.HasPrefix(k, key+"/") {
						delete(f.files, k)
					}
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}
			if _, ok := f.files[key]; !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			delete(f.files, key)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}

func newSeaweed(t *testing.T) (store.FileStore, *fakeFiler, *httptest.Server) {
	t.Helper()
	f := newFakeFiler()
	srv := httptest.NewServer(f.handler())
	s, err := store.NewSeaweedFS(srv.URL, swPrefix, swTimeout)
	require.Nil(t, err, fmt.Sprintf("new seaweedfs failed: %s", err))
	return s, f, srv
}

func TestSeaweedPut(t *testing.T) {
	s, f, srv := newSeaweed(t)
	defer srv.Close()

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
			desc:    "put empty file",
			key:     "groups/g3/empty.txt",
			content: []byte{},
			err:     nil,
		},
		{
			desc:    "put binary payload",
			key:     "groups/g4/b.bin",
			content: bytes.Repeat([]byte{0x00, 0xff}, 1024),
			err:     nil,
		},
	}

	for _, tc := range cases {
		checksum, err := s.Put(context.Background(), tc.key, bytes.NewReader(tc.content))
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))
		assert.Equal(t, sum(tc.content), checksum, fmt.Sprintf("%s: checksum mismatch", tc.desc))

		got := f.files[swPrefix+"/"+tc.key]
		assert.Equal(t, tc.content, got, fmt.Sprintf("%s: filer content mismatch", tc.desc))
	}
}

func TestSeaweedGet(t *testing.T) {
	s, _, srv := newSeaweed(t)
	defer srv.Close()

	content := []byte(testContent)
	expected, err := s.Put(context.Background(), testKey, bytes.NewReader(content))
	require.Nil(t, err, fmt.Sprintf("put failed: %s", err))

	cases := []struct {
		desc     string
		key      string
		checksum string
		want     []byte
		errMatch string
	}{
		{
			desc:     "get with valid checksum",
			key:      testKey,
			checksum: expected,
			want:     content,
		},
		{
			desc:     "get without checksum",
			key:      testKey,
			checksum: "",
			want:     content,
		},
		{
			desc:     "get missing key returns error",
			key:      "groups/missing/x.txt",
			checksum: "",
			errMatch: "404",
		},
	}

	for _, tc := range cases {
		rc, err := s.Get(context.Background(), tc.key, tc.checksum)
		if tc.errMatch != "" {
			assert.Error(t, err, fmt.Sprintf("%s: expected error", tc.desc))
			assert.Contains(t, err.Error(), tc.errMatch, fmt.Sprintf("%s: error mismatch", tc.desc))
			continue
		}
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))
		data, readErr := io.ReadAll(rc)
		require.Nil(t, readErr, fmt.Sprintf("%s: read failed: %s", tc.desc, readErr))
		rc.Close()
		assert.Equal(t, tc.want, data, fmt.Sprintf("%s: content mismatch", tc.desc))
	}
}

func TestSeaweedGetChecksumMismatch(t *testing.T) {
	s, _, srv := newSeaweed(t)
	defer srv.Close()

	_, err := s.Put(context.Background(), testKey, bytes.NewReader([]byte(testContent)))
	require.Nil(t, err, fmt.Sprintf("put failed: %s", err))

	rc, err := s.Get(context.Background(), testKey, wrongSum)
	require.Nil(t, err, fmt.Sprintf("get open failed: %s", err))
	defer rc.Close()

	_, err = io.ReadAll(rc)
	assert.True(t, errors.Is(err, store.ErrChecksumMismatch), fmt.Sprintf("expected ErrChecksumMismatch got %s", err))
}

func TestSeaweedDelete(t *testing.T) {
	s, f, srv := newSeaweed(t)
	defer srv.Close()

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

	_, exists := f.files[swPrefix+"/"+testKey]
	assert.False(t, exists, "file should be gone from filer")
}

func TestSeaweedDeleteAll(t *testing.T) {
	s, f, srv := newSeaweed(t)
	defer srv.Close()

	keys := []string{}
	for i := 0; i < 20; i++ {
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
			desc: "delete many in parallel",
			keys: keys,
			err:  nil,
		},
	}

	for _, tc := range cases {
		err := s.DeleteAll(context.Background(), tc.keys)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))
	}

	for _, k := range keys {
		_, exists := f.files[swPrefix+"/"+k]
		assert.False(t, exists, fmt.Sprintf("%s should be gone", k))
	}
}

func TestSeaweedDeletePrefix(t *testing.T) {
	s, f, srv := newSeaweed(t)
	defer srv.Close()

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
	}

	for _, tc := range cases {
		err := s.DeletePrefix(context.Background(), tc.prefix)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))

		for _, k := range tc.gone {
			_, exists := f.files[swPrefix+"/"+k]
			assert.False(t, exists, fmt.Sprintf("%s: %s should be gone", tc.desc, k))
		}
		for _, k := range tc.remains {
			_, exists := f.files[swPrefix+"/"+k]
			assert.True(t, exists, fmt.Sprintf("%s: %s should remain", tc.desc, k))
		}
	}
}
