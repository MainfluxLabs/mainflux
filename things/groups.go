package things

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

// Identity contains ID and Email.
type Identity struct {
	ID    string
	Email string
}

// Group represents the group information.
type Group struct {
	ID          string
	OrgID       string
	Name        string
	Description string
	Metadata    Metadata
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
	RetrieveByIDs(ctx context.Context, groupIDs []string, pm PageMetadata) (GroupPage, error)

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

	// ListProfilesByGroup retrieves page of profiles that are assigned to a group identified by ID.
	ListProfilesByGroup(ctx context.Context, token string, groupID string, pm PageMetadata) (ProfilesPage, error)

	// ViewGroupByThing retrieves group that thing belongs to.
	ViewGroupByThing(ctx context.Context, token, thingID string) (Group, error)

	// RemoveGroups removes the groups identified with the provided IDs.
	RemoveGroups(ctx context.Context, token string, ids ...string) error

	// ViewGroupByProfile retrieves group that profile belongs to.
	ViewGroupByProfile(ctx context.Context, token, profileID string) (Group, error)
}

// GroupCache contains group caching interface.
type GroupCache interface {
	// RemoveGroupEntities removes all entities related to the group identified by ID.
	RemoveGroupEntities(context.Context, string) error

	// SaveRole stores member's role for given group ID.
	SaveRole(context.Context, string, string, string) error

	// ViewRole returns a group member role by given groupID and memberID.
	ViewRole(context.Context, string, string) (string, error)

	// RemoveRole removes a group member role from cache.
	RemoveRole(context.Context, string, string) error

	// GroupMemberships returns the IDs of the groups the member belongs to.
	GroupMemberships(context.Context, string) ([]string, error)
}

func (ts *thingsService) CreateGroups(ctx context.Context, token string, groups ...Group) ([]Group, error) {
	user, err := ts.auth.Identify(ctx, &protomfx.Token{Value: token})
	if err != nil {
		return []Group{}, errors.Wrap(errors.ErrAuthentication, err)
	}
	userID := user.GetId()

	grs := []Group{}
	for _, group := range groups {
		oid, err := ts.auth.GetOwnerIDByOrgID(ctx, &protomfx.OrgID{Value: group.OrgID})
		if err != nil {
			return []Group{}, err
		}
		ownerID := oid.GetValue()

		members := []GroupMember{{MemberID: ownerID, Role: Owner}}
		if ownerID != userID {
			if err := ts.canAccessOrg(ctx, token, group.OrgID, auth.OrgSub, Editor); err != nil {
				return nil, err
			}
			members = append(members, GroupMember{MemberID: userID, Role: Admin})
		}

		gr, err := ts.createGroup(ctx, group)
		if err != nil {
			return []Group{}, err
		}

		for i := range members {
			members[i].GroupID = gr.ID
			if err := ts.roles.SaveRolesByGroup(ctx, members[i]); err != nil {
				return []Group{}, err
			}

			if err := ts.groupCache.SaveRole(ctx, gr.ID, members[i].MemberID, members[i].Role); err != nil {
				return []Group{}, err
			}
		}

		grs = append(grs, gr)
	}

	return grs, nil
}

func (ts *thingsService) createGroup(ctx context.Context, group Group) (Group, error) {
	timestamp := getTimestmap()
	group.CreatedAt, group.UpdatedAt = timestamp, timestamp

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
		if err := ts.canAccessOrg(ctx, token, orgID, auth.OrgSub, Viewer); err == nil {
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

	grIDs, err := ts.getGroupIDsByMemberID(ctx, user.GetId())
	if err != nil {
		return GroupPage{}, err
	}

	return ts.groups.RetrieveByIDs(ctx, grIDs, pm)
}

func (ts *thingsService) ListGroupsByIDs(ctx context.Context, ids []string) ([]Group, error) {
	page, err := ts.groups.RetrieveByIDs(ctx, ids, PageMetadata{})
	if err != nil {
		return []Group{}, err
	}

	return page.Groups, nil
}

func (ts *thingsService) RemoveGroups(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		ar := AuthorizeReq{
			Token:   token,
			Object:  id,
			Subject: GroupSub,
			Action:  Owner,
		}
		if err := ts.Authorize(ctx, ar); err != nil {
			return err
		}

		if err := ts.groupCache.RemoveGroupEntities(ctx, id); err != nil {
			return err
		}
	}

	return ts.groups.Remove(ctx, ids...)
}

func (ts *thingsService) UpdateGroup(ctx context.Context, token string, group Group) (Group, error) {
	ar := AuthorizeReq{
		Token:   token,
		Object:  group.ID,
		Subject: GroupSub,
		Action:  Admin,
	}
	if err := ts.Authorize(ctx, ar); err != nil {
		return Group{}, err
	}
	group.UpdatedAt = getTimestmap()

	return ts.groups.Update(ctx, group)
}

func (ts *thingsService) ViewGroup(ctx context.Context, token, groupID string) (Group, error) {
	ar := AuthorizeReq{
		Token:   token,
		Object:  groupID,
		Subject: GroupSub,
		Action:  Viewer,
	}
	if err := ts.Authorize(ctx, ar); err != nil {
		return Group{}, err
	}

	gr, err := ts.groups.RetrieveByID(ctx, groupID)
	if err != nil {
		return Group{}, err
	}

	return gr, nil
}

func (ts *thingsService) ViewGroupByProfile(ctx context.Context, token string, profileID string) (Group, error) {
	grID, err := ts.getGroupIDByProfileID(ctx, profileID)
	if err != nil {
		return Group{}, err
	}

	ar := AuthorizeReq{
		Token:   token,
		Object:  grID,
		Subject: GroupSub,
		Action:  Viewer,
	}
	if err := ts.Authorize(ctx, ar); err != nil {
		return Group{}, err
	}

	gr, err := ts.groups.RetrieveByID(ctx, grID)
	if err != nil {
		return Group{}, err
	}

	return gr, nil
}

func (ts *thingsService) ViewGroupByThing(ctx context.Context, token string, thingID string) (Group, error) {
	grID, err := ts.getGroupIDByThingID(ctx, thingID)
	if err != nil {
		return Group{}, err
	}

	ar := AuthorizeReq{
		Token:   token,
		Object:  grID,
		Subject: GroupSub,
		Action:  Viewer,
	}
	if err := ts.Authorize(ctx, ar); err != nil {
		return Group{}, err
	}

	gr, err := ts.groups.RetrieveByID(ctx, grID)
	if err != nil {
		return Group{}, err
	}

	return gr, nil
}

func (ts *thingsService) canAccessGroup(ctx context.Context, token, groupID, action string) error {
	if err := ts.isAdmin(ctx, token); err == nil {
		return nil
	}

	user, err := ts.auth.Identify(ctx, &protomfx.Token{Value: token})
	if err != nil {
		return err
	}

	gm := GroupMember{
		MemberID: user.Id,
		GroupID:  groupID,
	}

	role, err := ts.groupCache.ViewRole(ctx, gm.GroupID, gm.MemberID)
	if err != nil {
		r, err := ts.roles.RetrieveRole(ctx, gm)
		if err != nil {
			return err
		}
		role = r

		if err := ts.groupCache.SaveRole(ctx, gm.GroupID, gm.MemberID, r); err != nil {
			return err
		}
	}

	switch role {
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
		if action != Owner {
			return nil
		}
		return errors.ErrAuthorization
	case Owner:
		return nil
	default:
		return errors.ErrAuthorization
	}
}
