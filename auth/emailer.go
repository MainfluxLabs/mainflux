package auth

type Emailer interface {
	SendOrgInvite(To []string, inv OrgInvite, orgName string, invRedirectPath string) error
	SendPlatformInvite(To []string, inv PlatformInvite, redirectPath string) error
}
