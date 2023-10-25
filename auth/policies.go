package auth

import "context"

type GroupsPolicy struct {
	GroupID  string
	MemberID string
	Policy   string
}

type GroupInvitationByID struct {
	MemberID string
	Policy   string
}

type GroupInvitationByEmail struct {
	Email  string
	Policy string
}

type GroupMember struct {
	MemberID string
	Email    string
	Policy   string
}

type GroupMembersPage struct {
	PageMetadata
	GroupMembers []GroupMember
}

type Members interface {
	// CreateGroupMembers creates group members.
	CreateGroupMembers(ctx context.Context, token, groupID string, giByEmails ...GroupInvitationByEmail) error

	// ListGroupMembers retrieves page of group members.
	ListGroupMembers(ctx context.Context, token, groupID string, pm PageMetadata) (GroupMembersPage, error)

	// UpdateGroupMembers updates group members.
	UpdateGroupMembers(ctx context.Context, token, groupID string, giByEmails ...GroupInvitationByEmail) error

	// RemoveGroupMembers removes group members.
	RemoveGroupMembers(ctx context.Context, token, groupID string, memberIDs ...string) error
}

type MembersRepository interface {
	// SaveGroupMembers saves group members.
	SaveGroupMembers(ctx context.Context, groupID string, giByIDs ...GroupInvitationByID) error

	// RetrieveGroupMember retrieves group policy for a user.
	RetrieveGroupMember(ctc context.Context, gp GroupsPolicy) (string, error)

	// RetrieveGroupMembers retrieves page of group members.
	RetrieveGroupMembers(ctx context.Context, groupID string, pm PageMetadata) (GroupMembersPage, error)

	// RemoveGroupMembers removes group members.
	RemoveGroupMembers(ctx context.Context, groupID string, memberIDs ...string) error

	// UpdateGroupMembers updates group members.
	UpdateGroupMembers(ctx context.Context, groupID string, giByIDs ...GroupInvitationByID) error
}
