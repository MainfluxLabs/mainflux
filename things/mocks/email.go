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

func (e *emailerMock) SendGroupMembershipNotification(to []string, orgName, groupName, groupRole string) error {
	return nil
}
