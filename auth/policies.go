package auth

import "context"

type GroupsPolicy struct {
	GroupID  string
	MemberID string
	Policy   string
}

type GroupPolicyByID struct {
	MemberID string
	Policy   string
}

type GroupPolicyByEmail struct {
	Email  string
	Policy string
}

type GroupMemberPolicy struct {
	MemberID string
	Email    string
	Policy   string
}

type GroupPoliciesPage struct {
	PageMetadata
	GroupMembersPolicies []GroupMemberPolicy
}

type Members interface {
	// CreateGroupPolicies creates group policies.
	CreateGroupPolicies(ctx context.Context, token, groupID string, gps ...GroupPolicyByEmail) error

	// ListGroupPolicies retrieves page of group policies.
	ListGroupPolicies(ctx context.Context, token, groupID string, pm PageMetadata) (GroupPoliciesPage, error)

	// UpdateGroupPolicies updates group policies.
	UpdateGroupPolicies(ctx context.Context, token, groupID string, gps ...GroupPolicyByEmail) error

	// RemoveGroupPolicies removes group policies.
	RemoveGroupPolicies(ctx context.Context, token, groupID string, memberIDs ...string) error
}

type PoliciesRepository interface {
	// SaveGroupPolicies saves group policies.
	SaveGroupPolicies(ctx context.Context, groupID string, gps ...GroupPolicyByID) error

	// RetrieveGroupPolicy retrieves group policy.
	RetrieveGroupPolicy(ctc context.Context, gp GroupsPolicy) (string, error)

	// RetrieveGroupPolicies retrieves page of group policies.
	RetrieveGroupPolicies(ctx context.Context, groupID string, pm PageMetadata) (GroupPoliciesPage, error)

	// RemoveGroupPolicies removes group policies.
	RemoveGroupPolicies(ctx context.Context, groupID string, memberIDs ...string) error

	// UpdateGroupPolicies updates group policies.
	UpdateGroupPolicies(ctx context.Context, groupID string, gps ...GroupPolicyByID) error
}
