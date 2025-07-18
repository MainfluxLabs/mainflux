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
	uiResetURL       string
	uiEmailVerifyURL string
	agent            *email.Agent
}

// New creates new emailer utility
func New(uiResetURL, uiEmailVerifyURL string, c *email.Config) (users.Emailer, error) {
	e, err := email.New(c)
	return &emailer{
		uiResetURL:       uiResetURL,
		uiEmailVerifyURL: uiEmailVerifyURL,
		agent:            e,
	}, err
}

func (e *emailer) SendPasswordReset(To []string, uiHost string, token string) error {
	url := fmt.Sprintf("%s%s?token=%s", uiHost, e.uiResetURL, token)
	return e.agent.Send(To, "", "Password reset", "", url, "")
}

func (e *emailer) SendEmailVerification(To []string, uiHost, token string) error {
	subject := "Verify your MainfluxLabs e-mail address"
	content := `
		Use the following link to verify your e-mail address and complete registration:
		
		%s
	`

	url := fmt.Sprintf("%s%s?token=%s", uiHost, e.uiEmailVerifyURL, token)

	content = fmt.Sprintf(content, url)

	return e.agent.Send(To, "", subject, "", content, "")
}
