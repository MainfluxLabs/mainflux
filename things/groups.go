package things

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrAssignToGroup indicates failure to assign member to a group.
	ErrAssignToGroup = errors.New("failed to assign member to a group")

	// ErrGroupNotEmpty indicates group is not empty, can't be deleted.
	ErrGroupNotEmpty = errors.New("group is not empty")

	// ErrMemberAlreadyAssigned indicates that members is already assigned.
	ErrMemberAlreadyAssigned = errors.New("member is already assigned")

	// ErrFailedToRetrieveMembers failed to retrieve group members.
	ErrFailedToRetrieveMembers = errors.New("failed to retrieve group members")

	// ErrFailedToRetrieveMembership failed to retrieve memberships
	ErrFailedToRetrieveMembership = errors.New("failed to retrieve memberships")

	// ErrFailedToRetrieveAll failed to retrieve groups.
	ErrFailedToRetrieveAll = errors.New("failed to retrieve all groups")
)

// Identity contains ID and Email.
type Identity struct {
	ID    string
	Email string
}

// GroupMetadata defines the Metadata type.
type GroupMetadata map[string]interface{}

// Member represents the member information.
type Member struct {
	ID string
}

// Group represents the group information.
type Group struct {
	ID          string
	OwnerID     string
	Name        string
	Description string
	Metadata    GroupMetadata
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// GroupPage contains page related metadata as well as list of groups that
// belong to this page.
type GroupPage struct {
	PageMetadata
	Groups []Group
}

// MemberPage contains page related metadata as well as list of members that
// belong to this page.
type MemberPage struct {
	PageMetadata
	Members []string
}

// GroupRepository specifies a group persistence API.
type GroupRepository interface {
	// Save group
	Save(ctx context.Context, g Group) (Group, error)

	// Update a group
	Update(ctx context.Context, g Group) (Group, error)

	// Delete a group
	Delete(ctx context.Context, id string) error

	// RetrieveByID retrieves group by its id
	RetrieveByID(ctx context.Context, id string) (Group, error)

	// RetrieveAll retrieves all groups.
	RetrieveAll(ctx context.Context, pm PageMetadata) (GroupPage, error)

	//  Retrieves list of groups that member belongs to
	Memberships(ctx context.Context, memberID string, pm PageMetadata) (GroupPage, error)

	// Members retrieves everything that is assigned to a group identified by groupID.
	Members(ctx context.Context, groupID string, pm PageMetadata) (MemberPage, error)

	// Assign adds a member to group.
	Assign(ctx context.Context, groupID string, memberIDs ...string) error

	// Unassign removes a member from a group
	Unassign(ctx context.Context, groupID string, memberIDs ...string) error
}
