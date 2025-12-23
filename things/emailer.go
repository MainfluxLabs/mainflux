package things

type Emailer interface {
	SendGroupMembershipNotification(to []string, orgName, groupName, groupRole string) error
}
