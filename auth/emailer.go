package auth

type Emailer interface {
	SendOrgInvite(to []string, inv OrgInvite, orgName, invRedirectPath string) error
}
