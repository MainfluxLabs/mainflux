package things

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

// ErrGroupMembershipExists indicates that membership already exists.
var ErrGroupMembershipExists = errors.New("group membership already exists")

// ErrMemberNotInOrg indicates that the member does not belong to the organization of the group.
var ErrMemberNotInOrg = errors.New("member does not belong to the organization")

// Domain type aliases
type (
	GroupMembership      = domain.GroupMembership
	GroupMembershipsPage = domain.GroupMembershipsPage
)

// GroupMembershipsRepository specifies an interface for managing group memberships in persistence.
type GroupMembershipsRepository interface {
	// Save persists group memberships.
	Save(ctx context.Context, gms ...GroupMembership) error

	// RetrieveRole retrieves role of a specific group membership.
	RetrieveRole(ctx context.Context, gm GroupMembership) (string, error)

	// RetrieveByGroup retrieves a paginated list of group memberships by group ID.
	RetrieveByGroup(ctx context.Context, groupID string, pm PageMetadata) (GroupMembershipsPage, error)

	// BackupAll retrieves all group memberships. Used for backup.
	BackupAll(ctx context.Context) ([]GroupMembership, error)

	// RetrieveGroupIDsByMember retrieves IDs of groups where the member belongs.
	RetrieveGroupIDsByMember(ctx context.Context, memberID string) ([]string, error)

	// Remove removes the provided group memberships.
	Remove(ctx context.Context, groupID string, memberIDs ...string) error

	// Update updates existing group memberships.
	Update(ctx context.Context, gms ...GroupMembership) error
}

// GroupMemberships defines a service for managing group memberships.
type GroupMemberships interface {
	// CreateGroupMemberships adds memberships to a group identified by the provided ID.
	CreateGroupMemberships(ctx context.Context, token, redirectPath string, gms ...GroupMembership) error

	// CreateGroupMembershipsInternal saves group memberships without requiring authentication.
	CreateGroupMembershipsInternal(ctx context.Context, gms ...GroupMembership) error

	// ListGroupMemberships retrieves a paginated list of group memberships for the given group.
	ListGroupMemberships(ctx context.Context, token, groupID string, pm PageMetadata) (GroupMembershipsPage, error)

	// UpdateGroupMemberships updates roles of a specific group membership.
	UpdateGroupMemberships(ctx context.Context, token string, gms ...GroupMembership) error

	// RemoveGroupMemberships removes memberships from the given group.
	RemoveGroupMemberships(ctx context.Context, token, groupID string, memberIDs ...string) error

	// RemoveGroupMembershipsByOrg removes memberships of the members from all groups of the org.
	RemoveGroupMembershipsByOrg(ctx context.Context, orgID string, memberIDs ...string) error
}

func (ts *thingsService) CreateGroupMemberships(ctx context.Context, token, redirectPath string, gms ...GroupMembership) error {
	adminRolesByGroup := make(map[string]map[string]string)
	groupByID := make(map[string]Group)

	// Every membership is validated before any is saved, so a single rejected
	// member does not leave the earlier members of the batch persisted.
	for _, gm := range gms {
		ar := UserAccessReq{
			Token:  token,
			ID:     gm.GroupID,
			Action: Admin,
		}
		if err := ts.CanUserAccessGroup(ctx, ar); err != nil {
			return err
		}

		// Org owners and org admins hold their group role through their org role.
		adminRoles, ok := adminRolesByGroup[gm.GroupID]
		if !ok {
			roles, err := ts.getOrgAdminRoles(ctx, gm.GroupID)
			if err != nil {
				return err
			}
			adminRoles = roles
			adminRolesByGroup[gm.GroupID] = roles
		}

		if _, ok := adminRoles[gm.MemberID]; ok {
			return ErrGroupMembershipExists
		}

		group, ok := groupByID[gm.GroupID]
		if !ok {
			g, err := ts.groups.RetrieveByID(ctx, gm.GroupID)
			if err != nil {
				return err
			}
			group = g
			groupByID[gm.GroupID] = g
		}

		om, err := ts.auth.GetOrgMembership(ctx, group.OrgID, gm.MemberID)
		if err != nil {
			return err
		}
		if om.Role == "" {
			return ErrMemberNotInOrg
		}
	}

	if err := ts.groupMemberships.Save(ctx, gms...); err != nil {
		return err
	}

	for _, gm := range gms {
		if err := ts.groupCache.SaveGroupMembership(ctx, gm.GroupID, gm.MemberID, gm.Role); err != nil {
			return err
		}
	}

	for _, gm := range gms {
		group := groupByID[gm.GroupID]

		org, err := ts.auth.ViewOrg(ctx, token, group.OrgID)
		if err != nil {
			continue
		}

		page, err := ts.users.GetUsersByIDs(ctx, []string{gm.MemberID}, domain.UsersPageMetadata{})
		if err != nil {
			continue
		}

		recipientEmail := page.Users[0].Email

		// Send e-mail notification
		go func() {
			ts.email.SendGroupMembershipNotification([]string{recipientEmail}, org.Name, group.Name, gm.Role, redirectPath)
		}()
	}

	return nil
}

func (ts *thingsService) CreateGroupMembershipsInternal(ctx context.Context, gms ...GroupMembership) error {
	for _, gm := range gms {
		if err := ts.groupMemberships.Save(ctx, gm); err != nil {
			return err
		}

		if err := ts.groupCache.SaveGroupMembership(ctx, gm.GroupID, gm.MemberID, gm.Role); err != nil {
			return err
		}
	}

	return nil
}

func (ts *thingsService) ListGroupMemberships(ctx context.Context, token, groupID string, pm PageMetadata) (GroupMembershipsPage, error) {
	ar := UserAccessReq{
		Token:  token,
		ID:     groupID,
		Action: Viewer,
	}

	if err := ts.CanUserAccessGroup(ctx, ar); err != nil {
		return GroupMembershipsPage{}, err
	}

	gmp, err := ts.groupMemberships.RetrieveByGroup(ctx, groupID, PageMetadata{})
	if err != nil {
		return GroupMembershipsPage{}, err
	}

	adminRoles, err := ts.getOrgAdminRoles(ctx, groupID)
	if err != nil {
		return GroupMembershipsPage{}, err
	}

	size := len(gmp.GroupMemberships) + len(adminRoles)
	membershipByMemberID := make(map[string]GroupMembership, size)
	for _, m := range gmp.GroupMemberships {
		membershipByMemberID[m.MemberID] = m
	}

	// The org role of a member takes precedence over a stored group membership.
	for memberID, role := range adminRoles {
		membershipByMemberID[memberID] = GroupMembership{
			MemberID: memberID,
			Role:     role,
		}
	}

	memberIDs := make([]string, 0, len(membershipByMemberID))
	for memberID, m := range membershipByMemberID {
		if pm.Role == "" || m.Role == pm.Role {
			memberIDs = append(memberIDs, memberID)
		}
	}

	if len(memberIDs) == 0 {
		return GroupMembershipsPage{GroupMemberships: []GroupMembership{}}, nil
	}

	userPM := domain.UsersPageMetadata{
		Email:  pm.Email,
		Order:  pm.Order,
		Dir:    pm.Dir,
		Limit:  pm.Limit,
		Offset: pm.Offset,
	}

	page, err := ts.users.GetUsersByIDs(ctx, memberIDs, userPM)
	if err != nil {
		return GroupMembershipsPage{}, err
	}

	gms := []GroupMembership{}
	for _, u := range page.Users {
		if m, ok := membershipByMemberID[u.ID]; ok {
			m.Email = u.Email
			gms = append(gms, m)
		}
	}

	return GroupMembershipsPage{
		GroupMemberships: gms,
		Total:            page.Total,
	}, nil
}

func (ts *thingsService) UpdateGroupMemberships(ctx context.Context, token string, gms ...GroupMembership) error {
	adminRolesByGroup := make(map[string]map[string]string)
	for _, gm := range gms {
		ar := UserAccessReq{
			Token:  token,
			ID:     gm.GroupID,
			Action: Admin,
		}

		if err := ts.CanUserAccessGroup(ctx, ar); err != nil {
			return err
		}

		adminRoles, ok := adminRolesByGroup[gm.GroupID]
		if !ok {
			roles, err := ts.getOrgAdminRoles(ctx, gm.GroupID)
			if err != nil {
				return err
			}
			adminRoles = roles
			adminRolesByGroup[gm.GroupID] = roles
		}

		if _, ok := adminRoles[gm.MemberID]; ok {
			return errors.ErrAuthorization
		}

		if _, err := ts.groupCache.ViewRole(ctx, gm.GroupID, gm.MemberID); err != nil {
			if _, err := ts.groupMemberships.RetrieveRole(ctx, gm); err != nil {
				return err
			}
		}
	}

	if err := ts.groupMemberships.Update(ctx, gms...); err != nil {
		return err
	}

	for _, gm := range gms {
		if err := ts.groupCache.SaveGroupMembership(ctx, gm.GroupID, gm.MemberID, gm.Role); err != nil {
			return err
		}
	}

	return nil
}

func (ts *thingsService) RemoveGroupMemberships(ctx context.Context, token, groupID string, memberIDs ...string) error {
	ar := UserAccessReq{
		Token:  token,
		ID:     groupID,
		Action: Admin,
	}

	if err := ts.CanUserAccessGroup(ctx, ar); err != nil {
		return err
	}

	adminRoles, err := ts.getOrgAdminRoles(ctx, groupID)
	if err != nil {
		return err
	}

	for _, mid := range memberIDs {
		if _, ok := adminRoles[mid]; ok {
			return errors.ErrAuthorization
		}

		if _, err := ts.groupCache.ViewRole(ctx, groupID, mid); err != nil {
			gm := GroupMembership{
				GroupID:  groupID,
				MemberID: mid,
			}
			if _, err := ts.groupMemberships.RetrieveRole(ctx, gm); err != nil {
				return err
			}
		}
	}

	if err := ts.groupMemberships.Remove(ctx, groupID, memberIDs...); err != nil {
		return err
	}

	for _, mid := range memberIDs {
		if err := ts.groupCache.RemoveGroupMembership(ctx, groupID, mid); err != nil {
			return err
		}
	}

	return nil
}

func (ts *thingsService) RemoveGroupMembershipsByOrg(ctx context.Context, orgID string, memberIDs ...string) error {
	for _, mid := range memberIDs {
		grIDs, err := ts.groups.RetrieveIDsByOrgMembership(ctx, orgID, mid)
		if err != nil {
			return err
		}

		for _, grID := range grIDs {
			// The cache entry is removed first because a retry reads group IDs from the database.
			if err := ts.groupCache.RemoveGroupMembership(ctx, grID, mid); err != nil {
				return err
			}

			if err := ts.groupMemberships.Remove(ctx, grID, mid); err != nil {
				return err
			}
		}
	}

	return nil
}
