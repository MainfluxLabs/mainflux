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
	// SavePoliciesByGroup saves group policies by group ID.
	SavePoliciesByGroup(ctx context.Context, groupID string, gps ...GroupPolicyByID) error

	// RetrievePolicyByGroup retrieves group policy by group ID.
	RetrievePolicyByGroup(ctc context.Context, gp GroupPolicy) (string, error)

	// RetrievePoliciesByGroup retrieves page of group policies by groupID.
	RetrievePoliciesByGroup(ctx context.Context, groupID string, pm PageMetadata) (GroupPoliciesPage, error)

	// RetrieveAllPoliciesByGroup retrieves all group policies by group ID. This is used for backup.
	RetrieveAllPoliciesByGroup(ctx context.Context) ([]GroupPolicy, error)

	// RemovePoliciesByGroup removes group policies by group ID.
	RemovePoliciesByGroup(ctx context.Context, groupID string, memberIDs ...string) error

	// UpdatePoliciesByGroup updates group policies by group ID.
	UpdatePoliciesByGroup(ctx context.Context, groupID string, gps ...GroupPolicyByID) error
}

type Policies interface {
	// CreatePoliciesByGroup creates policies of the group identified by the provided ID.
	CreatePoliciesByGroup(ctx context.Context, token, groupID string, gps ...GroupPolicyByID) error

	// ListPoliciesByGroup retrieves a page of policies for a group that is identified by the provided ID.
	ListPoliciesByGroup(ctx context.Context, token, groupID string, pm PageMetadata) (GroupPoliciesPage, error)

	// UpdatePoliciesByGroup updates policies of the group identified by the provided ID.
	UpdatePoliciesByGroup(ctx context.Context, token, groupID string, gps ...GroupPolicyByID) error

	// RemovePoliciesByGroup removes policies of the group identified by the provided ID.
	RemovePoliciesByGroup(ctx context.Context, token, groupID string, memberIDs ...string) error
}

func (ts *thingsService) CreatePoliciesByGroup(ctx context.Context, token, groupID string, gps ...GroupPolicyByID) error {
	if err := ts.canAccessGroup(ctx, token, groupID, ReadWrite); err != nil {
		return err
	}

	if err := ts.policies.SavePoliciesByGroup(ctx, groupID, gps...); err != nil {
		return err
	}

	return nil
}

func (ts *thingsService) ListPoliciesByGroup(ctx context.Context, token, groupID string, pm PageMetadata) (GroupPoliciesPage, error) {
	if err := ts.canAccessGroup(ctx, token, groupID, Read); err != nil {
		return GroupPoliciesPage{}, err
	}

	gpp, err := ts.policies.RetrievePoliciesByGroup(ctx, groupID, pm)
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

func (ts *thingsService) UpdatePoliciesByGroup(ctx context.Context, token, groupID string, gps ...GroupPolicyByID) error {
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

	if err := ts.policies.UpdatePoliciesByGroup(ctx, groupID, gps...); err != nil {
		return err
	}

	return nil
}

func (ts *thingsService) RemovePoliciesByGroup(ctx context.Context, token, groupID string, memberIDs ...string) error {
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

	if err := ts.policies.RemovePoliciesByGroup(ctx, groupID, memberIDs...); err != nil {
		return err
	}

	return nil
}
