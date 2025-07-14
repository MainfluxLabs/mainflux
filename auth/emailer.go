package auth

type Emailer interface {
	SendOrgInvite(To []string, inviteID, orgName, uiHost, uiInvitePath string) error
}
