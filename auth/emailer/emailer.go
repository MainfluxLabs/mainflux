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

// TODO: I don't think it's appropriate for this function to accept the last two parameters - they should
// probably be set once when the Emailer instance is created.
func (e *emailer) SendOrgInvite(To []string, inviteID, orgName, uiHost, uiInvitePath string) error {
	uiInviteViewURL := fmt.Sprintf("%s%s/%s", uiHost, uiInvitePath, inviteID)
	emailContent := fmt.Sprintf(`
		Hello,

		You've been invited to join the %s Organization.

		Use the following URL to view the invite:
		%s
	`, orgName, uiInviteViewURL)

	return e.agent.Send(To, "", subjectOrgInvite, "", emailContent, "")
}
