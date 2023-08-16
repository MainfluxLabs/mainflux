package things

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrAssignGroupThing indicates failure to assign thing to a group.
	ErrAssignGroupThing = errors.New("failed to assign thing to a group")

	// ErrUnassignGroupThing indicates failure to unassign thing from a group.
	ErrUnassignGroupThing = errors.New("failed to unassign thing from a group")

	// ErrAssignGroupChannel indicates failure to assign channel to a group.
	ErrAssignGroupChannel = errors.New("failed to assign channel to a group")

	// ErrUnassignGroupChannel indicates failure to unassign channel from a group.
	ErrUnassignGroupChannel = errors.New("failed to unassign channel from a group")

	// ErrGroupNotEmpty indicates group is not empty, can't be deleted.
	ErrGroupNotEmpty = errors.New("group is not empty")

	// ErrThingAlreadyAssigned indicates that thing is already assigned.
	ErrThingAlreadyAssigned = errors.New("thing is already assigned")

	// ErrChannelAlreadyAssigned indicates that thing is already assigned.
	ErrChannelAlreadyAssigned = errors.New("channel is already assigned")

	// ErrFailedToRetrieveGroupThings indicates failure to retrieve group things.
	ErrFailedToRetrieveGroupThings = errors.New("failed to retrieve group things")

	// ErrFailedToRetrieveGroupChannels indicates failure to retrieve group channels.
	ErrFailedToRetrieveGroupChannels = errors.New("failed to retrieve group channels")

	// ErrFailedToRetrieveThingMembership indicates failure to retrieve thing membership
	ErrFailedToRetrieveThingMembership = errors.New("failed to retrieve thing membership")

	// ErrFailedToRetrieveChannelMembership indicates failure to retrieve channel membership
	ErrFailedToRetrieveChannelMembership = errors.New("failed to retrieve channel membership")
)

// Identity contains ID and Email.
type Identity struct {
	ID    string
	Email string
}

// GroupMetadata defines the Metadata type.
type GroupMetadata map[string]interface{}

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

// GroupThingRelation represents a relation between a group and a thing.
type GroupThingRelation struct {
	GroupID   string
	ThingID   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// GroupChannelRelation represents a relation between a group and a channel.
type GroupChannelRelation struct {
	GroupID   string
	ChannelID string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// GroupPage contains page related metadata as well as list of groups that
// belong to this page.
type GroupPage struct {
	PageMetadata
	Groups []Group
}

// GroupThingsPage contains page related metadata as well as list of members that
// belong to this page.
type GroupThingsPage struct {
	PageMetadata
	Things []Thing
}

type GroupChannelsPage struct {
	PageMetadata
	Channels []Channel
}

// GroupRepository specifies a group persistence API.
type GroupRepository interface {
	// Save group
	Save(ctx context.Context, g Group) (Group, error)

	// Update a group
	Update(ctx context.Context, g Group) (Group, error)

	// Remove a group
	Remove(ctx context.Context, id string) error

	// RetrieveByID retrieves group by its id
	RetrieveByID(ctx context.Context, id string) (Group, error)

	// RetrieveByIDs retrieves groups by their ids
	RetrieveByIDs(ctx context.Context, groupIDs []string) (GroupPage, error)

	// RetrieveByOwner retrieves all groups.
	RetrieveByOwner(ctx context.Context, ownerID string, pm PageMetadata) (GroupPage, error)

	// RetrieveThingMembership retrieves group that thing belongs to.
	RetrieveThingMembership(ctx context.Context, thingID string) (string, error)

	// RetrieveChannelMembership retrieves group that channel belongs to.
	RetrieveChannelMembership(ctx context.Context, channelID string) (string, error)

	// RetrieveGroupThings retrieves page of things that are assigned to a group identified by groupID.
	RetrieveGroupThings(ctx context.Context, groupID string, pm PageMetadata) (GroupThingsPage, error)

	// RetrieveGroupChannels retrieves page of channels that are assigned to a group identified by groupID.
	RetrieveGroupChannels(ctx context.Context, groupID string, pm PageMetadata) (GroupChannelsPage, error)

	// AssignThing adds a thing to a group
	AssignThing(ctx context.Context, groupID string, thingIDs ...string) error

	// UnassignThing removes a thing from a group
	UnassignThing(ctx context.Context, groupID string, thingIDs ...string) error

	// AssignChannel assigns a channel to a group
	AssignChannel(ctx context.Context, groupID string, ids ...string) error

	// UnassignChannel unassigns a channel from a group
	UnassignChannel(ctx context.Context, groupID string, ids ...string) error

	// RetrieveAll retrieves all groups.
	RetrieveAll(ctx context.Context) ([]Group, error)

	// RetrieveByAdmin retrieves all groups with pagination.
	RetrieveByAdmin(ctx context.Context, pm PageMetadata) (GroupPage, error)

	// RetrieveAllThingRelations retrieves all thing relations.
	RetrieveAllThingRelations(ctx context.Context) ([]GroupThingRelation, error)
}
