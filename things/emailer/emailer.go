package emailer

import (
	"fmt"

	"github.com/MainfluxLabs/mainflux/pkg/email"
	"github.com/MainfluxLabs/mainflux/things"
)

const (
	subjectGroupInvite = "You've been invited to join a Group"
)

type emailer struct {
	host  string
	agent *email.Agent
}

func New(host string, config *email.Config) (things.Emailer, error) {
	agent, err := email.New(config)
	if err != nil {
		return nil, err
	}

	return &emailer{
		host:  host,
		agent: agent,
	}, nil
}

func (e *emailer) SendGroupInvite(to []string, inv things.GroupInvite, orgName, invRedirectPath string) error {
	redirectURL := fmt.Sprintf("%s%s/%s", e.host, invRedirectPath, inv.ID)

	templateData := map[string]any{
		"GroupName":  inv.GroupName,
		"OrgName":    orgName,
		"Role":       inv.InviteeRole,
		"InviteLink": redirectURL,
	}

	return e.agent.Send(to, "", subjectGroupInvite, "group_invite", templateData)
}
