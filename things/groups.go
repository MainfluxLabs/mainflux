package things

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
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
	Total  uint64
	Groups []Group
}

// GroupRepository specifies a group persistence API.
type GroupRepository interface {
	// Save persists groups.
	Save(ctx context.Context, g ...Group) ([]Group, error)

	// Update performs an update to the existing group.
	Update(ctx context.Context, g Group) (Group, error)

	// Remove removes groups by their IDs.
	Remove(ctx context.Context, ids ...string) error

	// RemoveByOrg removes groups by org ID.
	RemoveByOrg(ctx context.Context, orgID string) error

	// RetrieveByID retrieves a group by its ID.
	RetrieveByID(ctx context.Context, id string) (Group, error)

	// RetrieveByIDs retrieves groups by their IDs.
	RetrieveByIDs(ctx context.Context, ids []string, pm apiutil.PageMetadata) (GroupPage, error)

	// BackupAll retrieves all groups.
	BackupAll(ctx context.Context) ([]Group, error)

	// RetrieveIDsByOrgMembership retrieves group IDs by org membership.
	RetrieveIDsByOrgMembership(ctx context.Context, orgID, memberID string) ([]string, error)

	// RetrieveIDsByOrg retrieves all group IDs by org.
	RetrieveIDsByOrg(ctx context.Context, orgID string) ([]string, error)

	// RetrieveAll retrieves all groups with pagination.
	RetrieveAll(ctx context.Context, pm apiutil.PageMetadata) (GroupPage, error)
}

type Groups interface {
	// CreateGroups adds groups to the user identified by the provided key.
	CreateGroups(ctx context.Context, token, orgID string, groups ...Group) ([]Group, error)

	// UpdateGroup updates the group identified by the provided ID.
	UpdateGroup(ctx context.Context, token string, g Group) (Group, error)

	// ViewGroup retrieves data about the group identified by ID.
	ViewGroup(ctx context.Context, token, id string) (Group, error)

	// ViewGroupInternal retrieves data about the Group identified by ID, without requiring authentication.
	ViewGroupInternal(ctx context.Context, id string) (Group, error)

	// ListGroups retrieves page of all groups.
	ListGroups(ctx context.Context, token string, pm apiutil.PageMetadata) (GroupPage, error)

	// ListGroupsByOrg retrieves page of groups that are assigned to an org identified by ID.
	ListGroupsByOrg(ctx context.Context, token, orgID string, pm apiutil.PageMetadata) (GroupPage, error)

	// ListThingsByGroup retrieves page of things that are assigned to a group identified by ID.
	ListThingsByGroup(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (ThingsPage, error)

	// ListProfilesByGroup retrieves page of profiles that are assigned to a group identified by ID.
	ListProfilesByGroup(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (ProfilesPage, error)

	// ViewGroupByThing retrieves group that thing belongs to.
	ViewGroupByThing(ctx context.Context, token, thingID string) (Group, error)

	// RemoveGroups removes the groups identified with the provided IDs.
	RemoveGroups(ctx context.Context, token string, ids ...string) error

	// RemoveGroupsByOrg removes groups related to an org identified by org ID.
	RemoveGroupsByOrg(ctx context.Context, orgID string) ([]string, error)

	// ViewGroupByProfile retrieves group that profile belongs to.
	ViewGroupByProfile(ctx context.Context, token, profileID string) (Group, error)
}

// GroupCache contains group caching interface.
type GroupCache interface {
	// RemoveGroupEntities removes all entities related to the group identified by groupID.
	RemoveGroupEntities(context.Context, string) error

	// SaveGroupMembership stores role for given groupID and memberID.
	SaveGroupMembership(context.Context, string, string, string) error

	// ViewRole returns role for given groupID and memberID.
	ViewRole(context.Context, string, string) (string, error)

	// RemoveGroupMembership removes group membership for given groupID and memberID.
	RemoveGroupMembership(context.Context, string, string) error

	// RetrieveGroupIDsByMember returns group IDs for given memberID.
	RetrieveGroupIDsByMember(context.Context, string) ([]string, error)
}

func (ts *thingsService) CreateGroups(ctx context.Context, token, orgID string, groups ...Group) ([]Group, error) {
	user, err := ts.auth.Identify(ctx, &protomfx.Token{Value: token})
	if err != nil {
		return []Group{}, errors.Wrap(errors.ErrAuthentication, err)
	}
	userID := user.GetId()

	oid, err := ts.auth.GetOwnerIDByOrgID(ctx, &protomfx.OrgID{Value: orgID})
	if err != nil {
		return []Group{}, err
	}
	ownerID := oid.GetValue()

	memberships := []GroupMembership{{MemberID: ownerID, Role: Owner}}
	if ownerID != userID {
		if err := ts.canAccessOrg(ctx, token, orgID, auth.OrgSub, Editor); err != nil {
			return nil, err
		}
		memberships = append(memberships, GroupMembership{MemberID: userID, Role: Admin})
	}

	grs := []Group{}
	for _, group := range groups {
		timestamp := getTimestamp()
		group.CreatedAt, group.UpdatedAt = timestamp, timestamp

		id, err := ts.idProvider.ID()
		if err != nil {
			return []Group{}, err
		}
		group.ID = id
		group.OrgID = orgID

		grs = append(grs, group)
	}

	grs, err = ts.groups.Save(ctx, grs...)
	if err != nil {
		return []Group{}, err
	}

	for _, gr := range grs {
		for i := range memberships {
			memberships[i].GroupID = gr.ID
			if err := ts.groupMemberships.Save(ctx, memberships[i]); err != nil {
				return []Group{}, err
			}

			if err := ts.groupCache.SaveGroupMembership(ctx, gr.ID, memberships[i].MemberID, memberships[i].Role); err != nil {
				return []Group{}, err
			}
		}
	}

	return grs, nil
}

func (ts *thingsService) ListGroups(ctx context.Context, token string, pm apiutil.PageMetadata) (GroupPage, error) {
	if err := ts.isAdmin(ctx, token); err == nil {
		return ts.groups.RetrieveAll(ctx, pm)
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

func (ts *thingsService) ListGroupsByOrg(ctx context.Context, token, orgID string, pm apiutil.PageMetadata) (GroupPage, error) {
	grIDs, err := ts.GetGroupIDsByOrg(ctx, orgID, token)
	if err != nil {
		return GroupPage{}, err
	}

	return ts.groups.RetrieveByIDs(ctx, grIDs, pm)
}

func (ts *thingsService) RemoveGroups(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		ar := UserAccessReq{
			Token:  token,
			ID:     id,
			Action: Owner,
		}
		if err := ts.CanUserAccessGroup(ctx, ar); err != nil {
			return err
		}

		if err := ts.groupCache.RemoveGroupEntities(ctx, id); err != nil {
			return err
		}
	}

	return ts.groups.Remove(ctx, ids...)
}

func (ts *thingsService) RemoveGroupsByOrg(ctx context.Context, orgID string) ([]string, error) {
	ids, err := ts.groups.RetrieveIDsByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}

	if len(ids) == 0 {
		return ids, nil
	}

	for _, id := range ids {
		if err := ts.groupCache.RemoveGroupEntities(ctx, id); err != nil {
			return nil, err
		}
	}

	if err := ts.groups.RemoveByOrg(ctx, orgID); err != nil {
		return nil, err
	}

	return ids, nil
}

func (ts *thingsService) UpdateGroup(ctx context.Context, token string, group Group) (Group, error) {
	ar := UserAccessReq{
		Token:  token,
		ID:     group.ID,
		Action: Admin,
	}
	if err := ts.CanUserAccessGroup(ctx, ar); err != nil {
		return Group{}, err
	}
	group.UpdatedAt = getTimestamp()

	return ts.groups.Update(ctx, group)
}

func (ts *thingsService) viewGroup(ctx context.Context, groupID string) (Group, error) {
	gr, err := ts.groups.RetrieveByID(ctx, groupID)
	if err != nil {
		return Group{}, err
	}

	return gr, nil
}

func (ts *thingsService) ViewGroup(ctx context.Context, token, groupID string) (Group, error) {
	ar := UserAccessReq{
		Token:  token,
		ID:     groupID,
		Action: Viewer,
	}
	if err := ts.CanUserAccessGroup(ctx, ar); err != nil {
		return Group{}, err
	}

	return ts.viewGroup(ctx, groupID)
}

func (ts *thingsService) ViewGroupInternal(ctx context.Context, groupID string) (Group, error) {
	return ts.viewGroup(ctx, groupID)
}

func (ts *thingsService) ViewGroupByProfile(ctx context.Context, token, profileID string) (Group, error) {
	ar := UserAccessReq{
		Token:  token,
		ID:     profileID,
		Action: Viewer,
	}
	if err := ts.CanUserAccessProfile(ctx, ar); err != nil {
		return Group{}, err
	}

	grID, err := ts.getGroupIDByProfileID(ctx, profileID)
	if err != nil {
		return Group{}, err
	}

	gr, err := ts.groups.RetrieveByID(ctx, grID)
	if err != nil {
		return Group{}, err
	}

	return gr, nil
}

func (ts *thingsService) ViewGroupByThing(ctx context.Context, token string, thingID string) (Group, error) {
	ar := UserAccessReq{
		Token:  token,
		ID:     thingID,
		Action: Viewer,
	}
	if err := ts.CanUserAccessThing(ctx, ar); err != nil {
		return Group{}, err
	}

	grID, err := ts.getGroupIDByThingID(ctx, thingID)
	if err != nil {
		return Group{}, err
	}

	gr, err := ts.groups.RetrieveByID(ctx, grID)
	if err != nil {
		return Group{}, err
	}

	return gr, nil
}

func (ts *thingsService) canAccessGroup(ctx context.Context, token, groupID, action string) error {
	user, err := ts.auth.Identify(ctx, &protomfx.Token{Value: token})
	if err != nil {
		return err
	}

	gm := GroupMembership{
		MemberID: user.Id,
		GroupID:  groupID,
	}

	role, err := ts.groupCache.ViewRole(ctx, gm.GroupID, gm.MemberID)
	if err != nil {
		r, err := ts.groupMemberships.RetrieveRole(ctx, gm)
		if err != nil {
			// root admin check: if it isn't a group member but has all rights
			if isAdminErr := ts.isAdmin(ctx, token); isAdminErr == nil {
				return nil
			}
			return err
		}
		role = r
		_ = ts.groupCache.SaveGroupMembership(ctx, gm.GroupID, gm.MemberID, r)
	}

	switch role {
	case Viewer:
		if action == Viewer {
			return nil
		}
	case Editor:
		if action == Viewer || action == Editor {
			return nil
		}
	case Admin:
		if action != Owner {
			return nil
		}
	case Owner:
		return nil
	}

	return errors.ErrAuthorization
}
