// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"sync"
)

const dirPerm = 0o755

// ErrChecksumMismatch indicates stored bytes do not match expected SHA256.
var ErrChecksumMismatch = errors.New("filestore: checksum mismatch")

// NewLocal returns FileStore rooted at base directory on local disk.
func NewLocal(base string) FileStore {
	return &localStore{base: base}
}

type localStore struct {
	base string
}

func (l *localStore) path(key string) string {
	return filepath.Join(l.base, key)
}

func (l *localStore) Put(_ context.Context, key string, r io.Reader) (string, error) {
	p := l.path(key)
	if err := os.MkdirAll(filepath.Dir(p), dirPerm); err != nil {
		return "", err
	}

	f, err := os.Create(p)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(f, io.TeeReader(r, h)); err != nil {
		_ = os.Remove(p)
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func (l *localStore) Get(_ context.Context, key, expected string) (io.ReadCloser, error) {
	f, err := os.Open(l.path(key))
	if err != nil {
		return nil, err
	}

	if expected == "" {
		return f, nil
	}

	return &verifyReader{rc: f, h: sha256.New(), expected: expected}, nil
}

func (l *localStore) Delete(_ context.Context, key string) error {
	err := os.Remove(l.path(key))
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func (l *localStore) DeleteAll(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, workers)
	errCh := make(chan error, len(keys))
	for _, k := range keys {
		wg.Add(1)
		sem <- struct{}{}
		go func(k string) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := l.Delete(ctx, k); err != nil {
				errCh <- err
			}
		}(k)
	}
	wg.Wait()

	close(errCh)
	for e := range errCh {
		if e != nil {
			return e
		}
	}

	return nil
}

func (l *localStore) DeletePrefix(_ context.Context, prefix string) error {
	p := l.path(prefix)
	if err := os.RemoveAll(p); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

type verifyReader struct {
	rc       io.ReadCloser
	h        hash.Hash
	expected string
	done     bool
}

func (v *verifyReader) Read(p []byte) (int, error) {
	n, err := v.rc.Read(p)
	if n > 0 {
		v.h.Write(p[:n])
	}

	if errors.Is(err, io.EOF) && !v.done {
		v.done = true
		got := hex.EncodeToString(v.h.Sum(nil))
		if got != v.expected {
			return n, fmt.Errorf("%w: got %s, want %s", ErrChecksumMismatch, got, v.expected)
		}
	}

	return n, err
}

func (v *verifyReader) Close() error {
	return v.rc.Close()
}
