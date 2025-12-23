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

func (e *emailerMock) SendOrgInvite(to []string, inv auth.OrgInvite, orgName, invRedirectPath string, groupNames map[string]string) error {
	return nil
}
