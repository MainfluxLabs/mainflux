// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package users_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/stretchr/testify/assert"
)

func TestValidatePassword(t *testing.T) {
	const minLength = 15

	cases := map[string]struct {
		password string
		email    string
		err      error
	}{
		"valid passphrase": {
			password: "correct horse battery staple",
			email:    "user@example.com",
			err:      nil,
		},
		"password shorter than minimum length": {
			password: "short-pass",
			email:    "user@example.com",
			err:      users.ErrPasswordTooShort,
		},
		"password longer than maximum length": {
			password: strings.Repeat("a", users.MaxPasswordLength+1),
			email:    "user@example.com",
			err:      users.ErrPasswordTooLong,
		},
		"password longer than 72 bytes": {
			password: strings.Repeat("čć", 37),
			email:    "user@example.com",
			err:      nil,
		},
		"common password": {
			password: "1q2w3e4r5t6y7u8i",
			email:    "user@example.com",
			err:      users.ErrCommonPassword,
		},
		"common password in different case": {
			password: "1Q2W3E4R5T6Y7U8I",
			email:    "user@example.com",
			err:      users.ErrCommonPassword,
		},
		"password containing e-mail local part": {
			password: "John.Doe-secure-pass",
			email:    "john.doe@example.com",
			err:      users.ErrPasswordContainsEmail,
		},
		"password containing short e-mail local part": {
			password: "a-long-secure-passphrase",
			email:    "a@example.com",
			err:      nil,
		},
		"unicode password counted in code points": {
			password: "čćžšđčćžšđčćžšđ",
			email:    "user@example.com",
			err:      nil,
		},
		"minimum length below the lowest allowed value": {
			password: "short",
			email:    "user@example.com",
			err:      users.ErrPasswordTooShort,
		},
	}

	for desc, tc := range cases {
		length := minLength
		if desc == "minimum length below the lowest allowed value" {
			length = 0
		}
		err := users.ValidatePassword(tc.password, tc.email, length)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
		if tc.err != nil {
			assert.True(t, errors.Contains(err, errors.ErrPasswordFormat), fmt.Sprintf("%s: expected %s got %s\n", desc, errors.ErrPasswordFormat, err))
		}
	}
}
