package emailer

import (
	"github.com/MainfluxLabs/mainflux/pkg/email"
	"github.com/MainfluxLabs/mainflux/things"
)

const (
	subjectGroupMembership = "You've been added to a Group!"
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

func (e *emailer) SendGroupMembershipNotification(to []string, orgName, groupName, groupRole string) error {

	templateData := map[string]any{
		"GroupName": groupName,
		"OrgName":   orgName,
		"Role":      groupRole,
	}

	return e.agent.Send(to, "", subjectGroupMembership, "group_membership", templateData)
}
