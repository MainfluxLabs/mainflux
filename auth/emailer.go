package auth

type Emailer interface {
	SendOrgInvite(To []string, inv OrgInvite, orgName, invRedirectPath string) error
}
