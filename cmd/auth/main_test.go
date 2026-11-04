// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateSecret(t *testing.T) {
	cases := []struct {
		desc   string
		secret string
		valid  bool
	}{
		{
			desc:   "empty secret",
			secret: "",
			valid:  false,
		},
		{
			desc:   "published default secret",
			secret: "secret",
			valid:  false,
		},
		{
			desc:   "secret one byte below the minimum length",
			secret: "abcde" + strings.Repeat("f", minSecretBytes-6),
			valid:  false,
		},
		{
			desc:   "secret of the minimum length",
			secret: "abcde" + strings.Repeat("f", minSecretBytes-5),
			valid:  true,
		},
		{
			desc:   "secret with too few distinct characters",
			secret: "abcd" + strings.Repeat("a", minSecretBytes),
			valid:  false,
		},
		{
			desc:   "secret with the minimum distinct characters",
			secret: "abcde" + strings.Repeat("a", minSecretBytes),
			valid:  true,
		},
	}

	for _, tc := range cases {
		err := validateSecret(tc.secret)
		assert.Equal(t, tc.valid, err == nil, fmt.Sprintf("%s: expected valid %t got error %v", tc.desc, tc.valid, err))
	}
}
