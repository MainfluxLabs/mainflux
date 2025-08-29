// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/users"
)

var _ users.Hasher = (*hasherMock)(nil)

type hasherMock struct{}

// NewHasher creates "no-op" hasher for test purposes. This implementation will
// return secrets without changing them.
func NewHasher() users.Hasher {
	return &hasherMock{}
}

func (hm *hasherMock) Hash(pwd string) (string, error) {
	if pwd == "" {
		return "", apiutil.ErrMalformedEntity
	}
	return pwd, nil
}

func (hm *hasherMock) Compare(plain, hashed string) error {
	if plain != hashed {
		return errors.ErrAuthentication
	}

	return nil
}
