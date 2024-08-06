package things

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

type GroupMembers struct {
	GroupID  string
	MemberID string
	Email    string
	Role     string
}

type GroupRoles struct {
	MemberID string
	Role     string
}
type GroupRolesPage struct {
	PageMetadata
	GroupRoles []GroupMembers
}

type RolesRepository interface {
	// SaveRolesByGroup saves group roles by group ID.
	SaveRolesByGroup(ctx context.Context, groupID string, gps ...GroupRoles) error

	// RetrieveRole retrieves group role by group ID.
	RetrieveRole(ctc context.Context, gp GroupMembers) (string, error)

	// RetrieveRolesByGroup retrieves page of group roles by groupID.
	RetrieveRolesByGroup(ctx context.Context, groupID string, pm PageMetadata) (GroupRolesPage, error)

	// RetrieveAllRolesByGroup retrieves all group roles by group ID. This is used for backup.
	RetrieveAllRolesByGroup(ctx context.Context) ([]GroupMembers, error)

	// RemoveRolesByGroup removes group roles by group ID.
	RemoveRolesByGroup(ctx context.Context, groupID string, memberIDs ...string) error

	// UpdateRolesByGroup updates group roles by group ID.
	UpdateRolesByGroup(ctx context.Context, groupID string, gps ...GroupRoles) error
}

type Policies interface {
	// CreateRolesByGroup creates policies of the group identified by the provided ID.
	CreateRolesByGroup(ctx context.Context, token, groupID string, gps ...GroupRoles) error

	// ListRolesByGroup retrieves a page of policies for a group that is identified by the provided ID.
	ListRolesByGroup(ctx context.Context, token, groupID string, pm PageMetadata) (GroupRolesPage, error)

	// UpdateRolesByGroup updates policies of the group identified by the provided ID.
	UpdateRolesByGroup(ctx context.Context, token, groupID string, gps ...GroupRoles) error

	// RemoveRolesByGroup removes policies of the group identified by the provided ID.
	RemoveRolesByGroup(ctx context.Context, token, groupID string, memberIDs ...string) error
}

func (ts *thingsService) CreateRolesByGroup(ctx context.Context, token, groupID string, gps ...GroupRoles) error {
	if err := ts.canAccessGroup(ctx, token, groupID, Admin); err != nil {
		return err
	}

	if err := ts.roles.SaveRolesByGroup(ctx, groupID, gps...); err != nil {
		return err
	}

	return nil
}

func (ts *thingsService) ListRolesByGroup(ctx context.Context, token, groupID string, pm PageMetadata) (GroupRolesPage, error) {
	if err := ts.canAccessGroup(ctx, token, groupID, Viewer); err != nil {
		return GroupRolesPage{}, err
	}

	gpp, err := ts.roles.RetrieveRolesByGroup(ctx, groupID, pm)
	if err != nil {
		return GroupRolesPage{}, err
	}

	var memberIDs []string
	for _, gp := range gpp.GroupRoles {
		memberIDs = append(memberIDs, gp.MemberID)
	}

	var groupRoles []GroupMembers
	if len(gpp.GroupRoles) > 0 {
		usrReq := protomfx.UsersByIDsReq{Ids: memberIDs}
		up, err := ts.users.GetUsersByIDs(ctx, &usrReq)
		if err != nil {
			return GroupRolesPage{}, err
		}

		emails := make(map[string]string)
		for _, user := range up.Users {
			emails[user.Id] = user.GetEmail()
		}

		for _, gp := range gpp.GroupRoles {
			email, ok := emails[gp.MemberID]
			if !ok {
				return GroupRolesPage{}, err
			}

			groupMember := GroupMembers{
				MemberID: gp.MemberID,
				Email:    email,
				Role:     gp.Role,
			}

			groupRoles = append(groupRoles, groupMember)
		}
	}

	page := GroupRolesPage{
		GroupRoles: groupRoles,
		PageMetadata: PageMetadata{
			Total:  gpp.Total,
			Offset: gpp.Offset,
			Limit:  gpp.Limit,
		},
	}

	return page, nil
}

func (ts *thingsService) UpdateRolesByGroup(ctx context.Context, token, groupID string, gps ...GroupRoles) error {
	if err := ts.canAccessGroup(ctx, token, groupID, Admin); err != nil {
		return err
	}

	group, err := ts.groups.RetrieveByID(ctx, groupID)
	if err != nil {
		return err
	}

	for _, gp := range gps {
		if gp.MemberID == group.OrgID {
			return errors.ErrAuthorization
		}
	}

	if err := ts.roles.UpdateRolesByGroup(ctx, groupID, gps...); err != nil {
		return err
	}

	return nil
}

func (ts *thingsService) RemoveRolesByGroup(ctx context.Context, token, groupID string, memberIDs ...string) error {
	if err := ts.canAccessGroup(ctx, token, groupID, Admin); err != nil {
		return err
	}

	group, err := ts.groups.RetrieveByID(ctx, groupID)
	if err != nil {
		return err
	}

	for _, m := range memberIDs {
		if m == group.OwnerID {
			return errors.ErrAuthorization
		}
	}

	if err := ts.roles.RemoveRolesByGroup(ctx, groupID, memberIDs...); err != nil {
		return err
	}

	return nil
}
