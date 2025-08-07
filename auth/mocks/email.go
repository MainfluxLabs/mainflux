// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import "github.com/MainfluxLabs/mainflux/auth"

type emailerMock struct {
}

// NewEmailer provides emailer instance for  the test
func NewEmailer() auth.Emailer {
	return &emailerMock{}
}

func (e *emailerMock) SendPasswordReset([]string, string, string) error {
	return nil
}

func (e *emailerMock) SendOrgInvite(To []string, inv auth.Invite, orgName string, invRedirectPath string, registerRedirectPath string) error {
	return nil
}
