// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"strings"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/users"
)

// LegacyHashPrefix marks hashes created by the legacy hasher mock.
const LegacyHashPrefix = "legacy:"

var (
	_ users.Hasher = (*hasherMock)(nil)
	_ users.Hasher = (*legacyHasherMock)(nil)
)

type hasherMock struct{}

// NewHasher creates "no-op" hasher for test purposes. This implementation will
// return secrets without changing them.
func NewHasher() users.Hasher {
	return &hasherMock{}
}

func (hm *hasherMock) Hash(pwd string) (string, error) {
	if pwd == "" {
		return "", errors.ErrMalformedEntity
	}
	return pwd, nil
}

func (hm *hasherMock) Compare(plain, hashed string) error {
	if strings.HasPrefix(hashed, LegacyHashPrefix) {
		return users.ErrUnsupportedHash
	}

	if plain != hashed {
		return errors.ErrAuthentication
	}

	return nil
}

type legacyHasherMock struct{}

// NewLegacyHasher creates a hasher for test purposes that marks secrets with LegacyHashPrefix.
func NewLegacyHasher() users.Hasher {
	return &legacyHasherMock{}
}

func (hm *legacyHasherMock) Hash(pwd string) (string, error) {
	return LegacyHashPrefix + pwd, nil
}

func (hm *legacyHasherMock) Compare(plain, hashed string) error {
	if LegacyHashPrefix+plain != hashed {
		return errors.ErrAuthentication
	}

	return nil
}
