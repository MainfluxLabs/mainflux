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

func (e *emailer) SendPasswordReset(To []string, path string, token string) error {
	url := fmt.Sprintf("%s%s?token=%s", e.host, path, token)
	return e.agent.Send(To, "", "Password reset", "", url, "")
}

func (e *emailer) SendEmailVerification(To []string, path string, token string) error {
	subject := "Verify your MainfluxLabs e-mail address"
	content := `
		Use the following link to verify your e-mail address and complete registration:
		
		%s
	`

	url := fmt.Sprintf("%s%s?token=%s", e.host, path, token)

	content = fmt.Sprintf(content, url)

	return e.agent.Send(To, "", subject, "", content, "")
}
