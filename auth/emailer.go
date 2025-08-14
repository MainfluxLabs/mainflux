package auth

type Emailer interface {
	SendOrgInvite(To []string, inv Invite, orgName string, invRedirectPath string, registerRedirectPath string) error
}
