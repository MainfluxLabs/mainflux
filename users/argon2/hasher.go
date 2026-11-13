// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package argon2 provides a hasher implementation utilizing Argon2id.
package argon2

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"runtime"
	"strings"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/users"
	"golang.org/x/crypto/argon2"
)

// OWASP minimum parameters for Argon2id.
const (
	memory      = 19 * 1024 // KiB
	iterations  = 2
	parallelism = 1
	saltLength  = 16
	keyLength   = 32

	prefix = "$argon2id$"
)

var (
	errHashPassword    = errors.New("generate hash from password failed")
	errComparePassword = errors.New("compare hash and password failed")
	errMalformedHash   = errors.New("malformed Argon2id hash")
)

var _ users.Hasher = (*argon2Hasher)(nil)

type argon2Hasher struct {
	// Each hash holds its memory until done, so running more hashes than CPUs only adds memory.
	slots chan struct{}
}

// New instantiates an Argon2id-based hasher implementation.
func New() users.Hasher {
	return &argon2Hasher{slots: make(chan struct{}, runtime.GOMAXPROCS(0))}
}

func (ah *argon2Hasher) Hash(password string) (string, error) {
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", errors.Wrap(errHashPassword, err)
	}

	key := ah.key(password, salt, iterations, memory, parallelism, keyLength)

	return fmt.Sprintf("%sv=%d$m=%d,t=%d,p=%d$%s$%s",
		prefix,
		argon2.Version,
		memory,
		iterations,
		parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

func (ah *argon2Hasher) Compare(password, hash string) error {
	if !strings.HasPrefix(hash, prefix) {
		return users.ErrUnsupportedHash
	}

	var version int
	var mem, iter uint32
	var par uint8
	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		return errors.Wrap(errComparePassword, errMalformedHash)
	}
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return errors.Wrap(errComparePassword, errMalformedHash)
	}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &mem, &iter, &par); err != nil {
		return errors.Wrap(errComparePassword, errMalformedHash)
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return errors.Wrap(errComparePassword, errMalformedHash)
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return errors.Wrap(errComparePassword, errMalformedHash)
	}

	key := ah.key(password, salt, iter, mem, par, uint32(len(expected)))
	if subtle.ConstantTimeCompare(key, expected) != 1 {
		return errComparePassword
	}

	return nil
}

func (ah *argon2Hasher) key(password string, salt []byte, iter, mem uint32, par uint8, length uint32) []byte {
	ah.slots <- struct{}{}
	defer func() { <-ah.slots }()

	return argon2.IDKey([]byte(password), salt, iter, mem, par, length)
}
