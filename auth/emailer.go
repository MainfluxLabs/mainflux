package auth

type Emailer interface {
	SendOrgInvite(To []string, inviteID, orgName, roleName, redirectPath string) error
}
