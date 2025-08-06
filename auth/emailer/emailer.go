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

func (e *emailer) SendOrgInvite(To []string, inviteID, orgName, roleName, redirectPath string) error {
	uiInviteViewURL := fmt.Sprintf("%s%s/%s", e.host, redirectPath, inviteID)
	emailContent := fmt.Sprintf(`
		Hello,

		You've been invited to join the %s Organization with role: %s.

		Use the following URL to view the invite:
		%s
	`, orgName, roleName, uiInviteViewURL)

	return e.agent.Send(To, "", subjectOrgInvite, "", emailContent, "")
}
