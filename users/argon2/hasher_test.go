// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package argon2_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/MainfluxLabs/mainflux/users/argon2"
	"github.com/MainfluxLabs/mainflux/users/bcrypt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const password = "correct horse battery staple"

func TestHash(t *testing.T) {
	hasher := argon2.New()

	first, err := hasher.Hash(password)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	second, err := hasher.Hash(password)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	assert.True(t, strings.HasPrefix(first, "$argon2id$v=19$m=19456,t=2,p=1$"), fmt.Sprintf("unexpected hash format: %s", first))
	assert.Greater(t, len(first), 60, "Argon2id hash expected to be longer than a bcrypt hash")
	assert.NotEqual(t, first, second, "hashes of the same password expected to use different salts")
}

func TestCompare(t *testing.T) {
	hasher := argon2.New()

	hash, err := hasher.Hash(password)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	legacyHash, err := bcrypt.New().Hash(password)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	parts := strings.Split(hash, "$")
	corruptedHash := strings.Join(append(parts[:5], "not-base64!"), "$")
	longPassword := strings.Repeat("čć", 64)
	longHash, err := hasher.Hash(longPassword)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc     string
		password string
		hash     string
		err      error
	}{
		{
			desc:     "compare matching password",
			password: password,
			hash:     hash,
			err:      nil,
		},
		{
			desc:     "compare wrong password",
			password: "wrong horse battery staple",
			hash:     hash,
			err:      errors.New("compare hash and password failed"),
		},
		{
			desc:     "compare long unicode password",
			password: longPassword,
			hash:     longHash,
			err:      nil,
		},
		{
			desc:     "compare with bcrypt hash",
			password: password,
			hash:     legacyHash,
			err:      users.ErrUnsupportedHash,
		},
		{
			desc:     "compare with empty hash",
			password: password,
			hash:     "",
			err:      users.ErrUnsupportedHash,
		},
		{
			desc:     "compare with malformed hash",
			password: password,
			hash:     corruptedHash,
			err:      errors.New("malformed Argon2id hash"),
		},
	}

	for _, tc := range cases {
		err := hasher.Compare(tc.password, tc.hash)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func BenchmarkArgon2Compare(b *testing.B) {
	hasher := argon2.New()
	hash, err := hasher.Hash(password)
	require.Nil(b, err, fmt.Sprintf("unexpected error: %s", err))

	for b.Loop() {
		_ = hasher.Compare(password, hash)
	}
}

func BenchmarkBcryptCompare(b *testing.B) {
	hasher := bcrypt.New()
	hash, err := hasher.Hash(password)
	require.Nil(b, err, fmt.Sprintf("unexpected error: %s", err))

	for b.Loop() {
		_ = hasher.Compare(password, hash)
	}
}
