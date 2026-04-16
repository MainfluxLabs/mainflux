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
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	workers = 8
)

type seaweedStore struct {
	base   string
	prefix string
	client *http.Client
}

// NewSeaweedFS returns FileStore backed by SeaweedFS filer HTTP API.
// filerURL example: http://filer:8888
// prefix is optional path namespace (e.g. "filestore").
func NewSeaweedFS(filerURL, prefix string, timeout time.Duration) (FileStore, error) {
	u, err := url.Parse(filerURL)
	if err != nil {
		return nil, err
	}

	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &seaweedStore{
		base:   strings.TrimRight(u.String(), "/"),
		prefix: strings.Trim(prefix, "/"),
		client: &http.Client{Timeout: timeout},
	}, nil
}

func (s *seaweedStore) url(key string) string {
	key = strings.TrimLeft(key, "/")
	if s.prefix != "" {
		key = s.prefix + "/" + key
	}
	return s.base + "/" + key
}

func (s *seaweedStore) Put(ctx context.Context, key string, r io.Reader) (string, error) {
	h := sha256.New()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url(key), io.TeeReader(r, h))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("seaweedfs put %s: %s", key, resp.Status)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func (s *seaweedStore) Get(ctx context.Context, key, expected string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.url(key), nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 300 {
		resp.Body.Close()
		return nil, fmt.Errorf("seaweedfs get %s: %s", key, resp.Status)
	}

	if expected == "" {
		return resp.Body, nil
	}

	return &verifyReader{rc: resp.Body, h: sha256.New(), expected: expected}, nil
}

func (s *seaweedStore) Delete(ctx context.Context, key string) error {
	return s.doDelete(ctx, s.url(key))
}

func (s *seaweedStore) DeletePrefix(ctx context.Context, prefix string) error {
	u := s.url(prefix) + "/?recursive=true"
	return s.doDelete(ctx, u)
}

func (s *seaweedStore) doDelete(ctx context.Context, u string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u, nil)
	if err != nil {
		return err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("seaweedfs delete: %s", resp.Status)
	}

	return nil
}

func (s *seaweedStore) DeleteAll(ctx context.Context, keys []string) error {
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
			if err := s.Delete(ctx, k); err != nil {
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
