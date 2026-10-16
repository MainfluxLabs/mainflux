// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"io"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

// ErrChecksumMismatch is returned when a retrieved file's SHA256 does not match the expected value.
var ErrChecksumMismatch = errors.New("checksum mismatch")

// ErrInvalidPrefix is returned when DeletePrefix is called with a prefix that
// would otherwise target the store root (empty or whitespace-only).
var ErrInvalidPrefix = errors.New("invalid prefix")

// ErrNotFound is returned by Get when the requested object does not exist.
var ErrNotFound = errors.New("object not found")

// limitedReadCloser reads from a limited view of an object and closes the object.
type limitedReadCloser struct {
	io.Reader
	io.Closer
}

// FileStore is a pluggable binary-object store.
type FileStore interface {
	// Put stores the given content under key and returns its SHA256 hex checksum and size in bytes.
	Put(ctx context.Context, key string, r io.Reader) (string, int64, error)

	// Get returns a streaming reader for key. If expectedChecksum is non-empty
	// the reader returns ErrChecksumMismatch at EOF when the digest differs.
	Get(ctx context.Context, key, expectedChecksum string) (io.ReadCloser, error)

	// GetRange returns a streaming reader for only part of the stored object,
	// starting at the given offset. The checksum is not verified, as it covers the whole object.
	GetRange(ctx context.Context, key string, offset, length int64) (io.ReadCloser, error)

	// Delete removes key.
	Delete(ctx context.Context, key string) error

	// DeletePrefix removes all objects whose key begins with prefix.
	DeletePrefix(ctx context.Context, prefix string) error
}
