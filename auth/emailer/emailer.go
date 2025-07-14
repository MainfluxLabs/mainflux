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
	agent *email.Agent
}

func New(emailConfig *email.Config) (auth.Emailer, error) {
	agent, err := email.New(emailConfig)

	return &emailer{
		agent: agent,
	}, err
}

func (e *emailer) SendOrgInvite(To []string, inv auth.Invite, orgName string, uiHost string, uiInvitePath string) error {
	uiInviteViewURL := fmt.Sprintf("%s%s/%s", uiHost, uiInvitePath, inv.ID)
	emailContent := fmt.Sprintf(`
		Hello,

		You've been invited to join the %s Organization.

		Use the following URL to view the invite:
		%s
	`, orgName, uiInviteViewURL)

	return e.agent.Send(To, "", subjectOrgInvite, "", emailContent, "")
}
