package auth

type Emailer interface {
	// Send the invitee an e-mail notifying them of the Org Invite. `groupNames` represents a mapping of
	// group IDs to group names for groups found in inv.Groups. It may be `nil` if and only if inv.Groups is also nil.
	SendOrgInvite(to []string, inv OrgInvite, orgName, invRedirectPath string, groupNames map[string]string) error
}
