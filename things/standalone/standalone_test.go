// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package standalone_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things/standalone"
	"github.com/stretchr/testify/assert"
)

const (
	email = "john.doe@example.com"
	token = "token"
)

func TestIdentify(t *testing.T) {
	svc := standalone.NewAuthService(email, token)

	cases := map[string]struct {
		token string
		id    string
		err   error
	}{
		"identify non-existing user": {
			token: "non-existing",
			id:    "",
			err:   errors.ErrAuthentication,
		},
		"identify existing user": {
			token: token,
			id:    email,
			err:   nil,
		},
	}

	for desc, tc := range cases {
		id, err := svc.Identify(context.Background(), tc.token)
		assert.Equal(t, tc.id, id.Email, fmt.Sprintf("%s: expected %s, got %s", desc, tc.id, id.Email))
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s, got %s", desc, tc.err, err))
	}
}

func TestIssue(t *testing.T) {
	svc := standalone.NewAuthService(email, token)

	cases := map[string]struct {
		token string
		id    string
		err   error
	}{
		"issue key with an invalid token": {
			token: "non-existing",
			id:    "",
			err:   errors.ErrAuthentication,
		},
		"issue key": {
			token: token,
			id:    token,
			err:   nil,
		},
	}

	for desc, tc := range cases {
		val, err := svc.Issue(context.Background(), tc.id, tc.token, 0)
		assert.Equal(t, tc.id, val, fmt.Sprintf("%s: expected %s, got %s", desc, tc.id, val))
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected %s, got %s", desc, tc.err, err))
	}
}
