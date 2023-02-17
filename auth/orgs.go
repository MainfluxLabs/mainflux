package auth

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrAssignToOrg indicates failure to assign member to a org.
	ErrAssignToOrg = errors.New("failed to assign member to a org")

	// ErrUnassignFromOrg indicates failure to unassign member from a org.
	ErrUnassignFromOrg = errors.New("failed to unassign member from a org")

	// ErrOrgNotEmpty indicates org is not empty, can't be deleted.
	ErrOrgNotEmpty = errors.New("org is not empty")

	// ErrMemberAlreadyAssigned indicates that members is already assigned.
	ErrOrgMemberAlreadyAssignedTo = errors.New("org member is already assigned")
)

// OrgMetadata defines the Metadata type.
type OrgMetadata map[string]interface{}

// Member represents the member information.
type OrgMember struct {
	ID string
}

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
type OrgPageMetadata struct {
	Total    uint64
	Offset   uint64
	Limit    uint64
	Name     string
	Metadata OrgMetadata
}

// OrgPage contains page related metadata as well as list of orgs that
// belong to this page.
type OrgPage struct {
	OrgPageMetadata
	Orgs []Org
}

// OrgMembersPage contains page related metadata as well as list of members that
// belong to this page.
type OrgMembersPage struct {
	OrgPageMetadata
	Members []OrgMember
}

// OrgService specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type OrgService interface {
	// CreateOrg creates new  org.
	CreateOrg(ctx context.Context, token string, g Org) (Org, error)

	// UpdateOrg updates the org identified by the provided ID.
	UpdateOrg(ctx context.Context, token string, g Org) (Org, error)

	// ViewOrg retrieves data about the org identified by ID.
	ViewOrg(ctx context.Context, token, id string) (Org, error)

	// ListOrgs retrieves orgs.
	ListOrgs(ctx context.Context, token string, pm OrgPageMetadata) (OrgPage, error)

	// ListOrgMembers retrieves everything that is assigned to a org identified by orgID.
	ListOrgMembers(ctx context.Context, token, orgID string, pm OrgPageMetadata) (OrgMembersPage, error)

	// ListOrgMemberships retrieves all orgs for member that is identified with memberID belongs to.
	ListOrgMemberships(ctx context.Context, token, memberID string, pm OrgPageMetadata) (OrgPage, error)

	// RemoveOrg removes the org identified with the provided ID.
	RemoveOrg(ctx context.Context, token, id string) error

	// AssignOrg adds a member with memberID into the org identified by orgID.
	AssignOrg(ctx context.Context, token, orgID string, memberIDs ...string) error

	// UnassignOrg removes member with memberID from org identified by orgID.
	UnassignOrg(ctx context.Context, token, orgID string, memberIDs ...string) error

	// AssignOrgAccessRights adds access rights on thing orgs to user org.
	AssignOrgAccessRights(ctx context.Context, token, thingOrgID, userOrgID string) error
}

// OrgRepository specifies a org persistence API.
type OrgRepository interface {
	// Save org
	Save(ctx context.Context, g Org) error

	// Update a org
	Update(ctx context.Context, g Org) error

	// Delete a org
	Delete(ctx context.Context, owner, id string) error

	// RetrieveByID retrieves org by its id
	RetrieveByID(ctx context.Context, id string) (Org, error)

	// RetrieveAll retrieves all orgs.
	RetrieveAll(ctx context.Context, ownerID string, pm OrgPageMetadata) (OrgPage, error)

	//  Retrieves list of orgs that member belongs to
	Memberships(ctx context.Context, memberID string, pm OrgPageMetadata) (OrgPage, error)

	// Members retrieves everything that is assigned to a org identified by orgID.
	Members(ctx context.Context, orgID string, pm OrgPageMetadata) (OrgMembersPage, error)

	// Assign adds a member to org.
	Assign(ctx context.Context, orgID string, memberIDs ...string) error

	// Unassign removes a member from a org
	Unassign(ctx context.Context, orgID string, memberIDs ...string) error
}
