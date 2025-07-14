package auth

type Emailer interface {
	SendOrgInvite(invite Invite, orgName string, uiHost string, uiInvitePath string) error
}
