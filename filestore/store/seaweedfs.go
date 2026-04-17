// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type seaweedFS struct {
	base   string
	prefix string
	client *http.Client
}

// NewSeaweedFS returns a FileStore backed by a SeaweedFS filer at url.
// prefix is prepended to every key (e.g. "filestore").
func NewSeaweedFS(url, prefix string, timeout time.Duration) (FileStore, error) {
	return &seaweedFS{
		base:   strings.TrimRight(url, "/"),
		prefix: strings.Trim(prefix, "/"),
		client: &http.Client{Timeout: timeout},
	}, nil
}

func (s *seaweedFS) objectURL(key string) string {
	return fmt.Sprintf("%s/%s/%s", s.base, s.prefix, key)
}

func (s *seaweedFS) Put(ctx context.Context, key string, r io.Reader) (string, error) {
	h := sha256.New()
	pr, pw := io.Pipe()
	go func() {
		_, err := io.Copy(pw, io.TeeReader(r, h))
		pw.CloseWithError(err)
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.objectURL(key), pr)
	if err != nil {
		return "", err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("seaweedfs put %s: status %d", key, resp.StatusCode)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (s *seaweedFS) Get(ctx context.Context, key, expectedChecksum string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.objectURL(key), nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("seaweedfs get %s: status %d", key, resp.StatusCode)
	}
	if expectedChecksum == "" {
		return resp.Body, nil
	}
	return &verifyReader{rc: resp.Body, h: sha256.New(), expected: expectedChecksum}, nil
}

func (s *seaweedFS) Delete(ctx context.Context, key string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, s.objectURL(key), nil)
	if err != nil {
		return err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusNoContent, http.StatusOK, http.StatusNotFound:
		return nil
	}
	return fmt.Errorf("seaweedfs delete %s: status %d", key, resp.StatusCode)
}

func (s *seaweedFS) DeleteAll(ctx context.Context, keys []string) error {
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
			if err := s.Delete(ctx, key); err != nil {
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

func (s *seaweedFS) DeletePrefix(ctx context.Context, prefix string) error {
	u := s.objectURL(prefix) + "/?recursive=true"
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusNoContent, http.StatusOK, http.StatusNotFound:
		return nil
	}
	return fmt.Errorf("seaweedfs delete prefix %s: status %d", prefix, resp.StatusCode)
}
