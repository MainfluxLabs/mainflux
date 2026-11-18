// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users

import "github.com/MainfluxLabs/mainflux/pkg/errors"

// ErrUnsupportedHash indicates that the hash was not created by the hasher.
var ErrUnsupportedHash = errors.New("unsupported hash format")

// Hasher specifies an API for generating hashes of an arbitrary textual
// content.
type Hasher interface {
	// Hash generates the hashed string from plain-text.
	Hash(string) (string, error)

	// Compare compares plain-text version to the hashed one. An error should
	// indicate failed comparison. ErrUnsupportedHash is returned if the
	// hash was not created by this hasher.
	Compare(string, string) error
}
