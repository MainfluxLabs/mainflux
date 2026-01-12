package emailer

import (
	"fmt"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/email"
)

const (
	subjectOrgInvite = "You're invited to join '%s'"
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

func (e *emailer) SendOrgInvite(to []string, inv auth.OrgInvite, orgName, invRedirectPath string, groupNames map[string]string) error {
	redirectURL := fmt.Sprintf("%s%s/%s", e.host, invRedirectPath, inv.ID)

	templateData := map[string]any{
		"OrgName":    orgName,
		"Role":       inv.InviteeRole,
		"InviteLink": redirectURL,
	}

	// If the Org invite is associated with one or more Group assignments, we build a mapping of group names to group roles using the passed-in
	// `groupNames` map and the group IDs from inv.Groups.
	if len(inv.Groups) > 0 {
		templateGroups := make(map[string]string, len(inv.Groups))
		for _, group := range inv.Groups {
			templateGroups[groupNames[group.GroupID]] = group.MemberRole
		}

		templateData["Groups"] = templateGroups
	}

	subject := fmt.Sprintf(subjectOrgInvite, orgName)
	return e.agent.Send(to, "", subject, "org_invite", templateData)
}
