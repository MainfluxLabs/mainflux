// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0
package emailer

import (
	"fmt"

	"github.com/MainfluxLabs/mainflux/pkg/email"
	"github.com/MainfluxLabs/mainflux/users"
)

const (
	subjectPlatformInvite    = "You've been invited to join Mainflux Labs"
	subjectPasswordReset     = "Password reset request"
	subjectEmailVerification = "Verify your e-mail address"
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
	templateData := map[string]any{
		"RedirectURL": url,
	}

	return e.agent.Send(To, "", subjectPasswordReset, "password_reset", templateData)
}

func (e *emailer) SendEmailVerification(To []string, redirectPath, token string) error {
	url := fmt.Sprintf("%s%s?token=%s", e.host, redirectPath, token)

	templateData := map[string]any{
		"RedirectURL": url,
	}

	return e.agent.Send(To, "", subjectEmailVerification, "email_verification", templateData)
}

func (e *emailer) SendPlatformInvite(to []string, inv users.PlatformInvite, redirectPath string) error {
	url := fmt.Sprintf("%s%s/%s", e.host, redirectPath, inv.ID)

	templateData := map[string]any{
		"RedirectURL": url,
	}

	return e.agent.Send(to, "", subjectPlatformInvite, "platform_invite", templateData)
}
