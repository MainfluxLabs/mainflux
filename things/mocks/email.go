package mocks

import (
	"github.com/MainfluxLabs/mainflux/things"
)

type emailerMock struct {
}

// NewEmailer provides emailer instance for  the test
func NewEmailer() things.Emailer {
	return &emailerMock{}
}

func (e *emailerMock) SendGroupInvite(to []string, inv things.GroupInvite, orgName, invRedirectPath string) error {
	return nil
}
