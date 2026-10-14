// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

// ErrBackend is returned for any non-2xx response from the object store.
// Concrete status + URL are logged by the caller, never surfaced.
var ErrBackend = errors.New("object store error")

const (
	httpScheme  = "http"
	httpsScheme = "https"
)

type seaweedFS struct {
	baseURL *url.URL
	prefix  string
	chunkMB int
	client  *http.Client
}

// NewSeaweedFS returns a FileStore backed by a SeaweedFS filer at rawURL.
// prefix is prepended to every key (e.g. "filestore"). timeout bounds the
// connect, TLS handshake, and response-header phases; the overall request
// deadline is driven by the caller's context so body transfers of large
// objects are not capped by a single whole-request timeout.
func NewSeaweedFS(rawURL, prefix string, timeout time.Duration, chunkMB int) (FileStore, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse seaweedfs url: %w", err)
	}

	if u.Scheme != httpScheme && u.Scheme != httpsScheme {
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
		chunkMB: chunkMB,
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

func (s *seaweedFS) Put(ctx context.Context, key string, r io.Reader) (string, int64, error) {
	h := sha256.New()
	pr, pw := io.Pipe()
	var size int64
	done := make(chan struct{})

	defer pr.Close()
	go func() {
		defer close(done)
		n, err := io.Copy(pw, io.TeeReader(r, h))
		size = n
		pw.CloseWithError(err)
	}()

	u, err := url.Parse(s.objectURL(key))
	if err != nil {
		return "", 0, err
	}
	u.RawQuery = url.Values{"maxMB": {strconv.Itoa(s.chunkMB)}}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), pr)
	if err != nil {
		return "", 0, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", 0, errors.Wrap(ErrBackend, fmt.Errorf("put status %d", resp.StatusCode))
	}

	// Wait for the copier so the hash and size are read only after it is done.
	pr.Close()
	<-done

	return hex.EncodeToString(h.Sum(nil)), size, nil
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
		if resp.StatusCode == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, errors.Wrap(ErrBackend, fmt.Errorf("get status %d", resp.StatusCode))
	}

	if expectedChecksum == "" {
		return resp.Body, nil
	}

	return &verifyReader{rc: resp.Body, h: sha256.New(), expected: expectedChecksum}, nil
}

func (s *seaweedFS) GetRange(ctx context.Context, key string, offset, length int64) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.objectURL(key), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", offset, offset+length-1))

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusPartialContent:
	case http.StatusOK:
		// The filer ignored the range and sent the whole object.
		if _, err := io.CopyN(io.Discard, resp.Body, offset); err != nil {
			resp.Body.Close()
			return nil, err
		}
	case http.StatusNotFound:
		resp.Body.Close()
		return nil, ErrNotFound
	default:
		resp.Body.Close()
		return nil, errors.Wrap(ErrBackend, fmt.Errorf("get range status %d", resp.StatusCode))
	}

	return limitedReadCloser{Reader: io.LimitReader(resp.Body, length), Closer: resp.Body}, nil
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

	return errors.Wrap(ErrBackend, fmt.Errorf("delete status %d", resp.StatusCode))
}

func (s *seaweedFS) DeletePrefix(ctx context.Context, prefix string) error {
	clean := strings.Trim(prefix, "/")
	if clean == "" {
		return ErrInvalidPrefix
	}

	u := *s.baseURL
	u.Path = path.Join(u.Path, s.prefix, clean) + "/"
	q := u.Query()
	q.Set("recursive", "true")
	q.Set("ignoreRecursiveError", "true")
	u.RawQuery = q.Encode()

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

	return errors.Wrap(ErrBackend, fmt.Errorf("delete prefix status %d", resp.StatusCode))
}
