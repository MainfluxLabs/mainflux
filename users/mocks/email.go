// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"github.com/MainfluxLabs/mainflux/users"
)

type emailerMock struct {
}

// NewEmailer provides emailer instance for  the test
func NewEmailer() users.Emailer {
	return &emailerMock{}
}

func (e *emailerMock) SendPasswordReset([]string, string, string) error {
	return nil
}

func (e *emailerMock) SendEmailVerification(To []string, uiHost, token string) error {
	return nil
}
