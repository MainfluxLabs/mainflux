// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"
)

// ErrBackend is returned for any non-2xx response from the object store.
// Concrete status + URL are logged by the caller, never surfaced.
var ErrBackend = errors.New("object store error")

type seaweedFS struct {
	baseURL *url.URL
	prefix  string
	client  *http.Client
}

// NewSeaweedFS returns a FileStore backed by a SeaweedFS filer at rawURL.
// prefix is prepended to every key (e.g. "filestore"). timeout bounds the
// connect, TLS handshake, and response-header phases; the overall request
// deadline is driven by the caller's context so body transfers of large
// objects are not capped by a single whole-request timeout.
func NewSeaweedFS(rawURL, prefix string, timeout time.Duration) (FileStore, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse seaweedfs url: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("invalid seaweedfs url scheme %q: want http or https", u.Scheme)
	}
	if u.Host == "" {
		return nil, fmt.Errorf("invalid seaweedfs url: %q", rawURL)
	}
	u.Path = strings.TrimRight(u.Path, "/")

	tr := &http.Transport{
		DialContext:           (&net.Dialer{Timeout: timeout, KeepAlive: 30 * time.Second}).DialContext,
		TLSHandshakeTimeout:   timeout,
		ResponseHeaderTimeout: timeout,
		ExpectContinueTimeout: 1 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          50,
		MaxIdleConnsPerHost:   10,
	}
	return &seaweedFS{
		baseURL: u,
		prefix:  strings.Trim(prefix, "/"),
		client:  &http.Client{Transport: tr},
	}, nil
}

// objectURL returns an RFC 3986-escaped URL for key. Each path segment is
// escaped so filenames with reserved characters (spaces, ?, #, &, etc.) do
// not alter the URL structure.
func (s *seaweedFS) objectURL(key string) string {
	u := *s.baseURL
	u.Path = path.Join(u.Path, s.prefix, key)
	return u.String()
}

func (s *seaweedFS) Put(ctx context.Context, key string, r io.Reader) (string, error) {
	h := sha256.New()
	pr, pw := io.Pipe()
	defer pr.Close()
	go func() {
		_, err := io.Copy(pw, io.TeeReader(r, h))
		pw.CloseWithError(err)
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, s.objectURL(key), pr)
	if err != nil {
		return "", err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: put status %d", ErrBackend, resp.StatusCode)
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
		return nil, fmt.Errorf("%w: get status %d", ErrBackend, resp.StatusCode)
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
	return fmt.Errorf("%w: delete status %d", ErrBackend, resp.StatusCode)
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
				errCh <- fmt.Errorf("key %s: %w", key, err)
			}
		}(k)
	}
	wg.Wait()
	close(errCh)
	var errs []error
	for e := range errCh {
		errs = append(errs, e)
	}
	return errors.Join(errs...)
}

func (s *seaweedFS) DeletePrefix(ctx context.Context, prefix string) error {
	clean := strings.Trim(prefix, "/")
	if clean == "" {
		return ErrInvalidPrefix
	}
	u := *s.baseURL
	u.Path = path.Join(u.Path, s.prefix, clean) + "/"
	u.RawQuery = "recursive=true&ignoreRecursiveError=true"

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u.String(), nil)
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
	return fmt.Errorf("%w: delete prefix status %d", ErrBackend, resp.StatusCode)
}
