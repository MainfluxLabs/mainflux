package auth

type Emailer interface {
	SendOrgInvite(To []string, inv Invite, orgName string, uiHost string, uiInvitePath string) error
}
