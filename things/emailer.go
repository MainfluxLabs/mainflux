package things

type Emailer interface {
	SendGroupInvite(to []string, inv GroupInvite, orgName, invRedirectPath string) error
}
