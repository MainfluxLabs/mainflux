package things

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

var (
	// ErrRetrieveGroupThings indicates failure to retrieve group things.
	ErrRetrieveGroupThings = errors.New("failed to retrieve group things")

	// ErrRetrieveGroupChannels indicates failure to retrieve group channels.
	ErrRetrieveGroupChannels = errors.New("failed to retrieve group channels")
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
	OrgID       string
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

// GroupRepository specifies a group persistence API.
type GroupRepository interface {
	// Save group
	Save(ctx context.Context, g Group) (Group, error)

	// Update a group
	Update(ctx context.Context, g Group) (Group, error)

	// Remove a groups
	Remove(ctx context.Context, groupIDs ...string) error

	// RetrieveByID retrieves group by its id
	RetrieveByID(ctx context.Context, id string) (Group, error)

	// RetrieveByIDs retrieves groups by their ids
	RetrieveByIDs(ctx context.Context, groupIDs []string) (GroupPage, error)

	// RetrieveByOwner retrieves all groups.
	RetrieveByOwner(ctx context.Context, ownerID, orgID string, pm PageMetadata) (GroupPage, error)

	// RetrieveThingsByGroup retrieves page of things that are assigned to a group identified by ID.
	RetrieveThingsByGroup(ctx context.Context, groupID string, pm PageMetadata) (ThingsPage, error)

	// RetrieveChannelsByGroup retrieves page of channels that are assigned to a group identified by ID.
	RetrieveChannelsByGroup(ctx context.Context, groupID string, pm PageMetadata) (ChannelsPage, error)

	// RetrieveAll retrieves all groups.
	RetrieveAll(ctx context.Context) ([]Group, error)

	// RetrieveByAdmin retrieves all groups with pagination.
	RetrieveByAdmin(ctx context.Context, orgID string, pm PageMetadata) (GroupPage, error)
}

type Groups interface {
	// CreateGroups adds groups to the user identified by the provided key.
	CreateGroups(ctx context.Context, token string, groups ...Group) ([]Group, error)

	// UpdateGroup updates the group identified by the provided ID.
	UpdateGroup(ctx context.Context, token string, g Group) (Group, error)

	// ViewGroup retrieves data about the group identified by ID.
	ViewGroup(ctx context.Context, token, id string) (Group, error)

	// ListGroups retrieves groups.
	ListGroups(ctx context.Context, token, orgID string, pm PageMetadata) (GroupPage, error)

	// ListGroupsByIDs retrieves groups by their IDs.
	ListGroupsByIDs(ctx context.Context, ids []string) ([]Group, error)

	// ListThingsByGroup retrieves page of things that are assigned to a group identified by ID.
	ListThingsByGroup(ctx context.Context, token string, groupID string, pm PageMetadata) (ThingsPage, error)

	// ListChannelsByGroup retrieves page of channels that are assigned to a group identified by ID.
	ListChannelsByGroup(ctx context.Context, token string, groupID string, pm PageMetadata) (ChannelsPage, error)

	// ViewGroupByThing retrieves group that thing belongs to.
	ViewGroupByThing(ctx context.Context, token, thingID string) (Group, error)

	// RemoveGroups removes the groups identified with the provided IDs.
	RemoveGroups(ctx context.Context, token string, ids ...string) error

	// ViewGroupByChannel retrieves group that channel belongs to.
	ViewGroupByChannel(ctx context.Context, token, channelID string) (Group, error)
}

func (ts *thingsService) CreateGroups(ctx context.Context, token string, groups ...Group) ([]Group, error) {
	user, err := ts.auth.Identify(ctx, &protomfx.Token{Value: token})
	if err != nil {
		return []Group{}, errors.Wrap(errors.ErrAuthentication, err)
	}

	ownerID := user.GetId()
	timestamp := getTimestmap()

	grs := []Group{}
	for _, group := range groups {
		group.OwnerID = ownerID
		group.CreatedAt = timestamp
		group.UpdatedAt = timestamp

		gr, err := ts.createGroup(ctx, group)
		if err != nil {
			return []Group{}, err
		}

		role := GroupRoles{
			MemberID: ownerID,
			Role:     Admin,
		}

		if err := ts.roles.SaveRolesByGroup(ctx, gr.ID, role); err != nil {
			return []Group{}, err
		}

		grs = append(grs, gr)
	}

	return grs, nil
}

func (ts *thingsService) createGroup(ctx context.Context, group Group) (Group, error) {
	id, err := ts.idProvider.ID()
	if err != nil {
		return Group{}, err
	}
	group.ID = id

	group, err = ts.groups.Save(ctx, group)
	if err != nil {
		return Group{}, err
	}

	return group, nil
}

func (ts *thingsService) ListGroups(ctx context.Context, token, orgID string, pm PageMetadata) (GroupPage, error) {
	if orgID != "" {
		if err := ts.canAccessOrg(ctx, token, orgID); err == nil {
			return ts.groups.RetrieveByAdmin(ctx, orgID, pm)
		}
	}

	if err := ts.isAdmin(ctx, token); err == nil {
		return ts.groups.RetrieveByAdmin(ctx, orgID, pm)
	}

	user, err := ts.auth.Identify(ctx, &protomfx.Token{Value: token})
	if err != nil {
		return GroupPage{}, err
	}

	return ts.groups.RetrieveByOwner(ctx, user.GetId(), orgID, pm)
}

func (ts *thingsService) ListGroupsByIDs(ctx context.Context, ids []string) ([]Group, error) {
	page, err := ts.groups.RetrieveByIDs(ctx, ids)
	if err != nil {
		return []Group{}, err
	}

	return page.Groups, nil
}

func (ts *thingsService) RemoveGroups(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		if err := ts.canAccessGroup(ctx, token, id, Admin); err != nil {
			return err
		}
	}

	return ts.groups.Remove(ctx, ids...)
}

func (ts *thingsService) UpdateGroup(ctx context.Context, token string, group Group) (Group, error) {
	if err := ts.canAccessGroup(ctx, token, group.ID, Admin); err != nil {
		return Group{}, err
	}

	group.UpdatedAt = getTimestmap()

	return ts.groups.Update(ctx, group)
}

func (ts *thingsService) ViewGroup(ctx context.Context, token, id string) (Group, error) {
	if err := ts.canAccessGroup(ctx, token, id, Viewer); err != nil {
		return Group{}, err
	}

	gr, err := ts.groups.RetrieveByID(ctx, id)
	if err != nil {
		return Group{}, err
	}

	return gr, nil
}

func (ts *thingsService) ViewGroupByChannel(ctx context.Context, token string, channelID string) (Group, error) {
	if _, err := ts.auth.Identify(ctx, &protomfx.Token{Value: token}); err != nil {
		return Group{}, err
	}

	ch, err := ts.channels.RetrieveByID(ctx, channelID)
	if err != nil {
		return Group{}, err
	}

	group, err := ts.groups.RetrieveByID(ctx, ch.GroupID)
	if err != nil {
		return Group{}, err
	}

	return group, nil
}

func (ts *thingsService) ViewGroupByThing(ctx context.Context, token string, thingID string) (Group, error) {
	if _, err := ts.auth.Identify(ctx, &protomfx.Token{Value: token}); err != nil {
		return Group{}, err
	}

	th, err := ts.things.RetrieveByID(ctx, thingID)
	if err != nil {
		return Group{}, err
	}

	group, err := ts.groups.RetrieveByID(ctx, th.GroupID)
	if err != nil {
		return Group{}, err
	}

	return group, nil
}

func (ts *thingsService) canAccessGroup(ctx context.Context, token, groupID, action string) error {
	if err := ts.isAdmin(ctx, token); err == nil {
		return nil
	}

	user, err := ts.auth.Identify(ctx, &protomfx.Token{Value: token})
	if err != nil {
		return err
	}

	gp := GroupMembers{
		MemberID: user.Id,
		GroupID:  groupID,
	}

	p, err := ts.roles.RetrieveRole(ctx, gp)
	if err != nil {
		return err
	}

	switch p {
	case Viewer:
		if action == Viewer {
			return nil
		}
		return errors.ErrAuthorization
	case Editor:
		if action == Viewer || action == Editor {
			return nil
		}
		return errors.ErrAuthorization
	case Admin:
		return nil
	default:
		return errors.ErrAuthorization
	}
}
