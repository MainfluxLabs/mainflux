package emailer

import (
	"fmt"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/email"
)

const (
	subjectOrgInvite      = "You've been invited to join an Organization"
	subjectPlatformInvite = "You've been invited to join MainfluxLabs"
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

func (e *emailer) SendOrgInvite(To []string, inv auth.OrgInvite, orgName string, invRedirectPath string) error {
	redirectURL := fmt.Sprintf("%s%s/%s", e.host, invRedirectPath, inv.ID)

	emailContent := fmt.Sprintf(`
		Hello,

		You've been invited to join the %s Organization with role: %s.

		Navigate to the following URL to view the invitation:
		%s
	`, orgName, inv.InviteeRole, redirectURL)

	return e.agent.Send(To, "", subjectOrgInvite, "", emailContent, "")
}

func (e *emailer) SendPlatformInvite(To []string, inv auth.PlatformInvite, redirectPath string) error {
	redirectURL := fmt.Sprintf("%s%s/%s", e.host, redirectPath, inv.ID)

	emailContent := fmt.Sprintf(`
		Hello,

		You've been invited to join the MainfluxLabs platform!

		Navigate to the following URL to create an account:
		%s
	`, redirectURL)

	return e.agent.Send(To, "", subjectPlatformInvite, "", emailContent, "")
}
