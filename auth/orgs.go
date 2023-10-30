package auth

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrAssignToOrg indicates failure to assign member to an org.
	ErrAssignToOrg = errors.New("failed to assign member to an org")

	// ErrUnassignFromOrg indicates failure to unassign member from an org.
	ErrUnassignFromOrg = errors.New("failed to unassign member from an org")

	// ErrOrgNotEmpty indicates org is not empty, can't be deleted.
	ErrOrgNotEmpty = errors.New("org is not empty")

	// ErrOrgMemberAlreadyAssigned indicates that members is already assigned.
	ErrOrgMemberAlreadyAssigned = errors.New("org member is already assigned")

	// ErrOrgGroupAlreadyAssigned indicates that group is already assigned.
	ErrOrgGroupAlreadyAssigned = errors.New("org group is already assigned")
)

// OrgMetadata defines the Metadata type.
type OrgMetadata map[string]interface{}

// Org represents the org information.
type Org struct {
	ID          string
	OwnerID     string
	Name        string
	Description string
	Metadata    OrgMetadata
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// PageMetadata contains page metadata that helps navigation.
type PageMetadata struct {
	Total    uint64
	Offset   uint64
	Limit    uint64
	Name     string
	Metadata OrgMetadata
}

// OrgsPage contains page related metadata as well as list of orgs that
// belong to this page.
type OrgsPage struct {
	PageMetadata
	Orgs []Org
}

// OrgMembersPage contains page related metadata as well as list of members that
// belong to this page.
type OrgMembersPage struct {
	PageMetadata
	OrgMembers []OrgMember
}

type User struct {
	ID     string
	Email  string
	Status string
}

type Group struct {
	ID          string
	OwnerID     string
	Name        string
	Description string
}

type GroupsPage struct {
	PageMetadata
	Groups []Group
}

type OrgGroupsPage struct {
	PageMetadata
	OrgGroups []OrgGroup
}

type OrgMember struct {
	MemberID  string
	OrgID     string
	Role      string
	CreatedAt time.Time
	UpdatedAt time.Time
	Email     string
}

type OrgGroup struct {
	GroupID   string
	OrgID     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Backup struct {
	Orgs          []Org
	OrgMembers    []OrgMember
	OrgGroups     []OrgGroup
	GroupPolicies []GroupPolicy
}

// Orgs specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Orgs interface {
	// CreateOrg creates new org.
	CreateOrg(ctx context.Context, token string, org Org) (Org, error)

	// UpdateOrg updates the org identified by the provided ID.
	UpdateOrg(ctx context.Context, token string, org Org) (Org, error)

	// ViewOrg retrieves data about the org identified by ID.
	ViewOrg(ctx context.Context, token, id string) (Org, error)

	// ListOrgs retrieves orgs.
	ListOrgs(ctx context.Context, token string, admin bool, pm PageMetadata) (OrgsPage, error)

	// ListOrgMemberships retrieves all orgs for member that is identified with memberID belongs to.
	ListOrgMemberships(ctx context.Context, token, memberID string, pm PageMetadata) (OrgsPage, error)

	// RemoveOrg removes the org identified with the provided ID.
	RemoveOrg(ctx context.Context, token, id string) error

	// AssignMembers adds members with member emails into the org identified by orgID.
	AssignMembers(ctx context.Context, token, orgID string, oms ...OrgMember) error

	// UnassignMembers removes members with member ids from org identified by orgID.
	UnassignMembers(ctx context.Context, token string, orgID string, memberIDs ...string) error

	// UpdateMembers updates members role in an org.
	UpdateMembers(ctx context.Context, token, orgID string, oms ...OrgMember) error

	// ListOrgMembers retrieves members assigned to an org identified by orgID.
	ListOrgMembers(ctx context.Context, token, orgID string, pm PageMetadata) (OrgMembersPage, error)

	// ViewMember retrieves member identified by memberID in org identified by orgID.
	ViewMember(ctx context.Context, token, orgID, memberID string) (OrgMember, error)

	// AssignGroups adds groups with groupIDs into the org identified by orgID.
	AssignGroups(ctx context.Context, token, orgID string, groupIDs ...string) error

	// UnassignGroups removes groups with groupIDs from org identified by orgID.
	UnassignGroups(ctx context.Context, token, orgID string, groupIDs ...string) error

	//ViewGroupMembership retrieves orgs where group is assigned.
	ViewGroupMembership(ctx context.Context, token, groupID string) (Org, error)

	// ListOrgGroups retrieves groups assigned to an org identified by orgID.
	ListOrgGroups(ctx context.Context, token, orgID string, pm PageMetadata) (GroupsPage, error)

	// Backup retrieves all orgs, org members and org groups. Only accessible by admin.
	Backup(ctx context.Context, token string) (Backup, error)

	// Restore adds orgs, org members and org groups from a backup. Only accessible by admin.
	Restore(ctx context.Context, token string, backup Backup) error
}

// OrgRepository specifies an org persistence API.
type OrgRepository interface {
	// Save orgs
	Save(ctx context.Context, orgs ...Org) error

	// Update an org
	Update(ctx context.Context, org Org) error

	// Delete an org
	Delete(ctx context.Context, owner, id string) error

	// RetrieveByID retrieves org by its id
	RetrieveByID(ctx context.Context, id string) (Org, error)

	// RetrieveByOwner retrieves orgs by owner.
	RetrieveByOwner(ctx context.Context, ownerID string, pm PageMetadata) (OrgsPage, error)

	// RetrieveAll retrieves all orgs.
	RetrieveAll(ctx context.Context) ([]Org, error)

	// RetrieveByAdmin retrieves all orgs with pagination.
	RetrieveByAdmin(ctx context.Context, pm PageMetadata) (OrgsPage, error)

	// RetrieveMemberships list of orgs that member belongs to
	RetrieveMemberships(ctx context.Context, memberID string, pm PageMetadata) (OrgsPage, error)

	// AssignMembers adds members to an org.
	AssignMembers(ctx context.Context, oms ...OrgMember) error

	// UnassignMembers removes members from an org
	UnassignMembers(ctx context.Context, orgID string, memberIDs ...string) error

	// UpdateMembers updates members role in an org.
	UpdateMembers(ctx context.Context, oms ...OrgMember) error

	// RetrieveRole retrieves role of member identified by memberID in org identified by orgID.
	RetrieveRole(ctx context.Context, memberID, orgID string) (string, error)

	// RetrieveMembers retrieves members assigned to an org identified by orgID.
	RetrieveMembers(ctx context.Context, orgID string, pm PageMetadata) (OrgMembersPage, error)

	// RetrieveAllOrgMembers retrieves all org members.
	RetrieveAllOrgMembers(ctx context.Context) ([]OrgMember, error)

	// AssignGroups adds groups to an org.
	AssignGroups(ctx context.Context, ogs ...OrgGroup) error

	// UnassignGroups removes groups from an org
	UnassignGroups(ctx context.Context, orgID string, groupIDs ...string) error

	// RetrieveGroups retrieves groups assigned to an org identified by orgID.
	RetrieveGroups(ctx context.Context, orgID string, pm PageMetadata) (OrgGroupsPage, error)

	// RetrieveByGroupID retrieves org where group is assigned.
	RetrieveByGroupID(ctx context.Context, groupID string) (Org, error)

	// RetrieveAllOrgGroups retrieves all org groups.
	RetrieveAllOrgGroups(ctx context.Context) ([]OrgGroup, error)
}
