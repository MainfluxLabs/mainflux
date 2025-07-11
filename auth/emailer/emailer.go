package emailer

import (
	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/email"
)

type emailer struct {
	agent *email.Agent
}

func New(emailConfig *email.Config) (auth.Emailer, error) {
	agent, err := email.New(emailConfig)

	return emailer{
		agent: agent,
	}, err
}
