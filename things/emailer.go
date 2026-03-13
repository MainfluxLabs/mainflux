package things

type Emailer interface {
	SendGroupMembershipNotification(to []string, orgName, groupName, groupRole, redirectPath string) error
}
