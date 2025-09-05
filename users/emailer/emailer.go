// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0
package emailer

import (
	"fmt"

	"github.com/MainfluxLabs/mainflux/pkg/email"
	"github.com/MainfluxLabs/mainflux/users"
)

const (
	subjectPlatformInvite = "You've been invited to join MainfluxLabs"
)

var _ users.Emailer = (*emailer)(nil)

type emailer struct {
	host  string
	agent *email.Agent
}

// New creates new emailer utility
func New(host string, c *email.Config) (users.Emailer, error) {
	e, err := email.New(c)
	if err != nil {
		return nil, err
	}

	return &emailer{
		agent: e,
		host:  host,
	}, nil
}

func (e *emailer) SendPasswordReset(To []string, redirectPath, token string) error {
	url := fmt.Sprintf("%s%s?token=%s", e.host, redirectPath, token)
	return e.agent.Send(To, "", "Password reset", "", url, "")
}

func (e *emailer) SendEmailVerification(To []string, redirectPath, token string) error {
	subject := "Verify your MainfluxLabs e-mail address"
	content := `
		Use the following link to verify your e-mail address and complete registration:
		
		%s
	`

	url := fmt.Sprintf("%s%s?token=%s", e.host, redirectPath, token)

	content = fmt.Sprintf(content, url)

	return e.agent.Send(To, "", subject, "", content, "")
}

func (e *emailer) SendPlatformInvite(To []string, inv users.PlatformInvite, redirectPath string) error {
	redirectURL := fmt.Sprintf("%s%s/%s", e.host, redirectPath, inv.ID)

	emailContent := fmt.Sprintf(`
		Hello,

		You've been invited to join the MainfluxLabs platform!

		Navigate to the following URL to create an account:
		%s
	`, redirectURL)

	return e.agent.Send(To, "", subjectPlatformInvite, "", emailContent, "")
}
