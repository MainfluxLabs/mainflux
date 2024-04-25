package things

import (
	"context"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

type GroupPolicy struct {
	GroupID  string
	MemberID string
	Email    string
	Policy   string
}

type GroupPolicyByID struct {
	MemberID string
	Policy   string
}
type GroupPoliciesPage struct {
	PageMetadata
	GroupPolicies []GroupPolicy
}

type PoliciesRepository interface {
	// SaveGroupPolicies saves group policies.
	SaveGroupPolicies(ctx context.Context, groupID string, gps ...GroupPolicyByID) error

	// RetrieveGroupPolicy retrieves group policy.
	RetrieveGroupPolicy(ctc context.Context, gp GroupPolicy) (string, error)

	// RetrieveGroupPolicies retrieves page of group policies.
	RetrieveGroupPolicies(ctx context.Context, groupID string, pm PageMetadata) (GroupPoliciesPage, error)

	// RetrieveAllGroupPolicies retrieves all group policies. This is used for backup.
	RetrieveAllGroupPolicies(ctx context.Context) ([]GroupPolicy, error)

	// RemoveGroupPolicies removes group policies.
	RemoveGroupPolicies(ctx context.Context, groupID string, memberIDs ...string) error

	// UpdateGroupPolicies updates group policies.
	UpdateGroupPolicies(ctx context.Context, groupID string, gps ...GroupPolicyByID) error
}

type Policies interface {
	// CreateGroupPolicies creates group policies.
	CreateGroupPolicies(ctx context.Context, token, groupID string, gps ...GroupPolicyByID) error

	// ListGroupPolicies retrieves page of group policies.
	ListGroupPolicies(ctx context.Context, token, groupID string, pm PageMetadata) (GroupPoliciesPage, error)

	// UpdateGroupPolicies updates group policies.
	UpdateGroupPolicies(ctx context.Context, token, groupID string, gps ...GroupPolicyByID) error

	// RemoveGroupPolicies removes group policies.
	RemoveGroupPolicies(ctx context.Context, token, groupID string, memberIDs ...string) error
}

func (ts *thingsService) CreateGroupPolicies(ctx context.Context, token, groupID string, gps ...GroupPolicyByID) error {
	if err := ts.canAccessGroup(ctx, token, groupID, ReadWrite); err != nil {
		return err
	}

	if err := ts.policies.SaveGroupPolicies(ctx, groupID, gps...); err != nil {
		return err
	}

	return nil
}

func (ts *thingsService) ListGroupPolicies(ctx context.Context, token, groupID string, pm PageMetadata) (GroupPoliciesPage, error) {
	if err := ts.canAccessGroup(ctx, token, groupID, Read); err != nil {
		return GroupPoliciesPage{}, err
	}

	gpp, err := ts.policies.RetrieveGroupPolicies(ctx, groupID, pm)
	if err != nil {
		return GroupPoliciesPage{}, err
	}

	var memberIDs []string
	for _, gp := range gpp.GroupPolicies {
		memberIDs = append(memberIDs, gp.MemberID)
	}

	var groupPolicies []GroupPolicy
	if len(gpp.GroupPolicies) > 0 {
		usrReq := mainflux.UsersByIDsReq{Ids: memberIDs}
		up, err := ts.users.GetUsersByIDs(ctx, &usrReq)
		if err != nil {
			return GroupPoliciesPage{}, err
		}

		emails := make(map[string]string)
		for _, user := range up.Users {
			emails[user.Id] = user.GetEmail()
		}

		for _, gp := range gpp.GroupPolicies {
			email, ok := emails[gp.MemberID]
			if !ok {
				return GroupPoliciesPage{}, err
			}

			groupPolicy := GroupPolicy{
				MemberID: gp.MemberID,
				Email:    email,
				Policy:   gp.Policy,
			}

			groupPolicies = append(groupPolicies, groupPolicy)
		}
	}

	page := GroupPoliciesPage{
		GroupPolicies: groupPolicies,
		PageMetadata: PageMetadata{
			Total:  gpp.Total,
			Offset: gpp.Offset,
			Limit:  gpp.Limit,
		},
	}

	return page, nil
}

func (ts *thingsService) UpdateGroupPolicies(ctx context.Context, token, groupID string, gps ...GroupPolicyByID) error {
	if err := ts.canAccessGroup(ctx, token, groupID, ReadWrite); err != nil {
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

	if err := ts.policies.UpdateGroupPolicies(ctx, groupID, gps...); err != nil {
		return err
	}

	return nil
}

func (ts *thingsService) RemoveGroupPolicies(ctx context.Context, token, groupID string, memberIDs ...string) error {
	if err := ts.canAccessGroup(ctx, token, groupID, ReadWrite); err != nil {
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

	if err := ts.policies.RemoveGroupPolicies(ctx, groupID, memberIDs...); err != nil {
		return err
	}

	return nil
}
