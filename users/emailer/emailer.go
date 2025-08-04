// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0
package emailer

import (
	"fmt"

	"github.com/MainfluxLabs/mainflux/pkg/email"
	"github.com/MainfluxLabs/mainflux/users"
)

var _ users.Emailer = (*emailer)(nil)

type emailer struct {
	resetURL       string
	emailVerifyURL string
	agent          *email.Agent
}

// New creates new emailer utility
func New(resetURL, emailVerifyURL string, c *email.Config) (users.Emailer, error) {
	e, err := email.New(c)
	return &emailer{
		resetURL:       resetURL,
		emailVerifyURL: emailVerifyURL,
		agent:          e,
	}, err
}

func (e *emailer) SendPasswordReset(To []string, host string, token string) error {
	url := fmt.Sprintf("%s%s?token=%s", host, e.resetURL, token)
	return e.agent.Send(To, "", "Password reset", "", url, "")
}

func (e *emailer) SendEmailVerification(To []string, host, token string) error {
	subject := "Verify your MainfluxLabs e-mail address"
	content := `
		Use the following link to verify your e-mail address and complete registration:
		
		%s
	`

	url := fmt.Sprintf("%s%s?token=%s", host, e.emailVerifyURL, token)

	content = fmt.Sprintf(content, url)

	return e.agent.Send(To, "", subject, "", content, "")
}
