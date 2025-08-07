package emailer

import (
	"fmt"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/email"
)

const (
	subjectOrgInvite = "You've been invited to join an Organization"
)

type emailer struct {
	host  string
	agent *email.Agent
}

func New(host string, config *email.Config) (auth.Emailer, error) {
	agent, err := email.New(config)
	if err != nil {
		return nil, err
	}

	return &emailer{
		host:  host,
		agent: agent,
	}, nil
}

func (e *emailer) SendOrgInvite(To []string, inv auth.Invite, orgName string, invRedirectPath string, registerRedirectPath string) error {
	var redirectURL, instruction string
	if inv.InviteeID != "" {
		redirectURL = fmt.Sprintf("%s%s/%s", e.host, invRedirectPath, inv.ID)
		instruction = "Navigate to the following URL to view the invitation:"
	} else {
		redirectURL = fmt.Sprintf("%s/%s", e.host, registerRedirectPath)
		instruction = "Navigate to the following URL to register a MainfluxLabs user account:"
	}

	emailContent := fmt.Sprintf(`
		Hello,

		You've been invited to join the %s Organization with role: %s.

		%s:
		%s
	`, orgName, inv.InviteeRole, instruction, redirectURL)

	return e.agent.Send(To, "", subjectOrgInvite, "", emailContent, "")
}
