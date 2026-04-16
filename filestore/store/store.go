// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"io"
)

type FileStore interface {
	// Put streams reader into key. Returns SHA256 hex checksum of stored bytes.
	Put(ctx context.Context, key string, r io.Reader) (string, error)

	// Get opens stored object for streaming read. Caller must Close.
	// expectedChecksum may be empty, if set, integrity verified after EOF.
	Get(ctx context.Context, key, expectedChecksum string) (io.ReadCloser, error)

	// Delete removes single key.
	Delete(ctx context.Context, key string) error

	// DeleteAll removes set of keys.
	DeleteAll(ctx context.Context, keys []string) error

	// DeletePrefix removes all keys under prefix.
	DeletePrefix(ctx context.Context, prefix string) error
}
