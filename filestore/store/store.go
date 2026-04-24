// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"errors"
	"io"
)

// ErrChecksumMismatch is returned when a retrieved file's SHA256 does not match the expected value.
var ErrChecksumMismatch = errors.New("checksum mismatch")

// ErrInvalidPrefix is returned when DeletePrefix is called with a prefix that
// would otherwise target the store root (empty or whitespace-only).
var ErrInvalidPrefix = errors.New("invalid prefix")

// FileStore is a pluggable binary-object store.
type FileStore interface {
	// Put streams r under key and returns the SHA256 hex checksum.
	Put(ctx context.Context, key string, r io.Reader) (string, error)

	// Get returns a streaming reader for key. If expectedChecksum is non-empty
	// the reader returns ErrChecksumMismatch at EOF when the digest differs.
	Get(ctx context.Context, key, expectedChecksum string) (io.ReadCloser, error)

	// Delete removes key. Missing key is a no-op.
	Delete(ctx context.Context, key string) error

	// DeleteAll removes each key in parallel. Missing keys are no-ops.
	DeleteAll(ctx context.Context, keys []string) error

	// DeletePrefix removes all objects whose key begins with prefix.
	DeletePrefix(ctx context.Context, prefix string) error
}
