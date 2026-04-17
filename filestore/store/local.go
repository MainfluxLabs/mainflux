// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type local struct {
	base string
}

// NewLocal returns a FileStore backed by the local filesystem rooted at base.
func NewLocal(base string) FileStore {
	return &local{base: base}
}

func (l *local) Put(_ context.Context, key string, r io.Reader) (string, error) {
	path := filepath.Join(l.base, key)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}

	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(f, io.TeeReader(r, h)); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func (l *local) Get(_ context.Context, key, expectedChecksum string) (io.ReadCloser, error) {
	f, err := os.Open(filepath.Join(l.base, key))
	if err != nil {
		return nil, err
	}

	if expectedChecksum == "" {
		return f, nil
	}

	return &verifyReader{rc: f, h: sha256.New(), expected: expectedChecksum}, nil
}

func (l *local) Delete(_ context.Context, key string) error {
	if err := os.Remove(filepath.Join(l.base, key)); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func (l *local) DeleteAll(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	errCh := make(chan error, len(keys))
	for _, k := range keys {
		wg.Add(1)
		sem <- struct{}{}
		go func(key string) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := l.Delete(ctx, key); err != nil {
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

func (l *local) DeletePrefix(_ context.Context, prefix string) error {
	if err := os.RemoveAll(filepath.Join(l.base, prefix)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// verifyReader wraps a ReadCloser and returns ErrChecksumMismatch at EOF
// when the SHA256 of the streamed bytes differs from expected.
type verifyReader struct {
	rc       io.ReadCloser
	h        hash.Hash
	expected string
}

func (v *verifyReader) Read(p []byte) (int, error) {
	n, err := v.rc.Read(p)
	if n > 0 {
		v.h.Write(p[:n])
	}
	if err == io.EOF {
		if got := hex.EncodeToString(v.h.Sum(nil)); got != v.expected {
			return n, ErrChecksumMismatch
		}
	}
	return n, err
}

func (v *verifyReader) Close() error {
	return v.rc.Close()
}
