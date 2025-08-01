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
	uiInviteURL string
	agent       *email.Agent
}

func New(uiInviteURL string, config *email.Config) (auth.Emailer, error) {
	agent, err := email.New(config)

	return &emailer{
		uiInviteURL: uiInviteURL,
		agent:       agent,
	}, err
}

func (e *emailer) SendOrgInvite(To []string, inviteID, orgName, roleName, uiHost string) error {
	uiInviteViewURL := fmt.Sprintf("%s%s/%s", uiHost, e.uiInviteURL, inviteID)
	emailContent := fmt.Sprintf(`
		Hello,

		You've been invited to join the %s Organization with role: %s.

		Use the following URL to view the invite:
		%s
	`, orgName, roleName, uiInviteViewURL)

	return e.agent.Send(To, "", subjectOrgInvite, "", emailContent, "")
}
