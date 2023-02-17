// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/ulid"
)

const (
	recoveryDuration = 5 * time.Minute
	thingsGroupType  = "things"

	authoritiesObject = "authorities"
	memberRelation    = "member"
)

var (
	// ErrFailedToRetrieveMembers failed to retrieve group members.
	ErrFailedToRetrieveMembers = errors.New("failed to retrieve group members")

	// ErrFailedToRetrieveMembership failed to retrieve memberships
	ErrFailedToRetrieveMembership = errors.New("failed to retrieve memberships")

	// ErrFailedToRetrieveAll failed to retrieve groups.
	ErrFailedToRetrieveAll = errors.New("failed to retrieve all groups")

	// ErrFailedToRetrieveParents failed to retrieve groups.
	ErrFailedToRetrieveParents = errors.New("failed to retrieve all groups")

	// ErrFailedToRetrieveChildren failed to retrieve groups.
	ErrFailedToRetrieveChildren = errors.New("failed to retrieve all groups")

	errIssueUser = errors.New("failed to issue new login key")
	errIssueTmp  = errors.New("failed to issue new temporary key")
	errRevoke    = errors.New("failed to remove key")
	errRetrieve  = errors.New("failed to retrieve key data")
	errIdentify  = errors.New("failed to validate token")
)

// Authn specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
// Token is a string value of the actual Key and is used to authenticate
// an Auth service request.
type Authn interface {
	// Issue issues a new Key, returning its token value alongside.
	Issue(ctx context.Context, token string, key Key) (Key, string, error)

	// Revoke removes the Key with the provided id that is
	// issued by the user identified by the provided key.
	Revoke(ctx context.Context, token, id string) error

	// RetrieveKey retrieves data for the Key identified by the provided
	// ID, that is issued by the user identified by the provided key.
	RetrieveKey(ctx context.Context, token, id string) (Key, error)

	// Identify validates token token. If token is valid, content
	// is returned. If token is invalid, or invocation failed for some
	// other reason, non-nil error value is returned in response.
	Identify(ctx context.Context, token string) (Identity, error)
}

// Service specifies an API that must be fulfilled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
// Token is a string value of the actual Key and is used to authenticate
// an Auth service request.
type Service interface {
	Authn
	Authz

	// GroupService implements groups API, creating groups, assigning members
	OrgService

	// GroupService implements groups API, creating groups, assigning members
	GroupService
}

var _ Service = (*service)(nil)

type service struct {
	orgs          OrgRepository
	keys          KeyRepository
	groups        GroupRepository
	idProvider    mainflux.IDProvider
	ulidProvider  mainflux.IDProvider
	tokenizer     Tokenizer
	loginDuration time.Duration
	adminEmail    string
}

// New instantiates the auth service implementation.
func New(orgs OrgRepository, keys KeyRepository, groups GroupRepository, idp mainflux.IDProvider, tokenizer Tokenizer, duration time.Duration, adminEmail string) Service {
	return &service{
		tokenizer:     tokenizer,
		orgs:          orgs,
		keys:          keys,
		groups:        groups,
		idProvider:    idp,
		ulidProvider:  ulid.New(),
		loginDuration: duration,
		adminEmail:    adminEmail,
	}
}

func (svc service) Issue(ctx context.Context, token string, key Key) (Key, string, error) {
	if key.IssuedAt.IsZero() {
		return Key{}, "", ErrInvalidKeyIssuedAt
	}
	switch key.Type {
	case APIKey:
		return svc.userKey(ctx, token, key)
	case RecoveryKey:
		return svc.tmpKey(recoveryDuration, key)
	default:
		return svc.tmpKey(svc.loginDuration, key)
	}
}

func (svc service) Revoke(ctx context.Context, token, id string) error {
	issuerID, _, err := svc.login(token)
	if err != nil {
		return errors.Wrap(errRevoke, err)
	}
	if err := svc.keys.Remove(ctx, issuerID, id); err != nil {
		return errors.Wrap(errRevoke, err)
	}
	return nil
}

func (svc service) RetrieveKey(ctx context.Context, token, id string) (Key, error) {
	issuerID, _, err := svc.login(token)
	if err != nil {
		return Key{}, errors.Wrap(errRetrieve, err)
	}

	return svc.keys.Retrieve(ctx, issuerID, id)
}

func (svc service) Identify(ctx context.Context, token string) (Identity, error) {
	key, err := svc.tokenizer.Parse(token)
	if err == ErrAPIKeyExpired {
		err = svc.keys.Remove(ctx, key.IssuerID, key.ID)
		return Identity{}, errors.Wrap(ErrAPIKeyExpired, err)
	}
	if err != nil {
		return Identity{}, errors.Wrap(errIdentify, err)
	}

	switch key.Type {
	case RecoveryKey, LoginKey:
		return Identity{ID: key.IssuerID, Email: key.Subject}, nil
	case APIKey:
		_, err := svc.keys.Retrieve(context.TODO(), key.IssuerID, key.ID)
		if err != nil {
			return Identity{}, errors.ErrAuthentication
		}
		return Identity{ID: key.IssuerID, Email: key.Subject}, nil
	default:
		return Identity{}, errors.ErrAuthentication
	}
}

func (svc service) Authorize(ctx context.Context, pr PolicyReq) error {
	if pr.Object == "authorities" && pr.Relation == "member" && pr.Subject != svc.adminEmail {
		return errors.ErrAuthorization
	}
	return nil
}

func (svc service) AddPolicy(ctx context.Context, pr PolicyReq) error {
	return nil
}

func (svc service) AddPolicies(ctx context.Context, token, object string, subjectIDs, relations []string) error {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return err
	}

	if err := svc.Authorize(ctx, PolicyReq{Object: authoritiesObject, Relation: memberRelation, Subject: user.ID}); err != nil {
		return err
	}

	var errs error
	for _, subjectID := range subjectIDs {
		for _, relation := range relations {
			if err := svc.AddPolicy(ctx, PolicyReq{Object: object, Relation: relation, Subject: subjectID}); err != nil {
				errs = errors.Wrap(fmt.Errorf("cannot add '%s' policy on object '%s' for subject '%s': %s", relation, object, subjectID, err), errs)
			}
		}
	}
	return errs
}

func (svc service) DeletePolicy(ctx context.Context, pr PolicyReq) error {
	return nil
}

func (svc service) DeletePolicies(ctx context.Context, token, object string, subjectIDs, relations []string) error {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return err
	}

	// Check if the user identified by token is the admin.
	if err := svc.Authorize(ctx, PolicyReq{Object: authoritiesObject, Relation: memberRelation, Subject: user.ID}); err != nil {
		return err
	}

	var errs error
	for _, subjectID := range subjectIDs {
		for _, relation := range relations {
			if err := svc.DeletePolicy(ctx, PolicyReq{Object: object, Relation: relation, Subject: subjectID}); err != nil {
				errs = errors.Wrap(fmt.Errorf("cannot delete '%s' policy on object '%s' for subject '%s': %s", relation, object, subjectID, err), errs)
			}
		}
	}
	return errs
}

func (svc service) AssignGroupAccessRights(ctx context.Context, token, thingGroupID, userGroupID string) error {
	if _, err := svc.Identify(ctx, token); err != nil {
		return err
	}
	return nil
}

func (svc service) ListPolicies(ctx context.Context, pr PolicyReq) (PolicyPage, error) {
	return PolicyPage{}, nil
}

func (svc service) tmpKey(duration time.Duration, key Key) (Key, string, error) {
	key.ExpiresAt = key.IssuedAt.Add(duration)
	secret, err := svc.tokenizer.Issue(key)
	if err != nil {
		return Key{}, "", errors.Wrap(errIssueTmp, err)
	}

	return key, secret, nil
}

func (svc service) userKey(ctx context.Context, token string, key Key) (Key, string, error) {
	id, sub, err := svc.login(token)
	if err != nil {
		return Key{}, "", errors.Wrap(errIssueUser, err)
	}

	key.IssuerID = id
	if key.Subject == "" {
		key.Subject = sub
	}

	keyID, err := svc.idProvider.ID()
	if err != nil {
		return Key{}, "", errors.Wrap(errIssueUser, err)
	}
	key.ID = keyID

	if _, err := svc.keys.Save(ctx, key); err != nil {
		return Key{}, "", errors.Wrap(errIssueUser, err)
	}

	secret, err := svc.tokenizer.Issue(key)
	if err != nil {
		return Key{}, "", errors.Wrap(errIssueUser, err)
	}

	return key, secret, nil
}

func (svc service) login(token string) (string, string, error) {
	key, err := svc.tokenizer.Parse(token)
	if err != nil {
		return "", "", err
	}
	// Only login key token is valid for login.
	if key.Type != LoginKey || key.IssuerID == "" {
		return "", "", errors.ErrAuthentication
	}

	return key.IssuerID, key.Subject, nil
}

func (svc service) CreateGroup(ctx context.Context, token string, group Group) (Group, error) {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return Group{}, err
	}

	ulid, err := svc.ulidProvider.ID()
	if err != nil {
		return Group{}, err
	}

	timestamp := getTimestmap()
	group.UpdatedAt = timestamp
	group.CreatedAt = timestamp

	group.ID = ulid
	group.OwnerID = user.ID

	group, err = svc.groups.Save(ctx, group)
	if err != nil {
		return Group{}, err
	}

	return group, nil
}

func (svc service) ListGroups(ctx context.Context, token string, pm PageMetadata) (GroupPage, error) {
	if _, err := svc.Identify(ctx, token); err != nil {
		return GroupPage{}, err
	}
	return svc.groups.RetrieveAll(ctx, pm)
}

func (svc service) ListParents(ctx context.Context, token string, childID string, pm PageMetadata) (GroupPage, error) {
	if _, err := svc.Identify(ctx, token); err != nil {
		return GroupPage{}, err
	}
	return svc.groups.RetrieveAllParents(ctx, childID, pm)
}

func (svc service) ListChildren(ctx context.Context, token string, parentID string, pm PageMetadata) (GroupPage, error) {
	if _, err := svc.Identify(ctx, token); err != nil {
		return GroupPage{}, err
	}
	return svc.groups.RetrieveAllChildren(ctx, parentID, pm)
}

func (svc service) ListMembers(ctx context.Context, token string, groupID, groupType string, pm PageMetadata) (MemberPage, error) {
	if _, err := svc.Identify(ctx, token); err != nil {
		return MemberPage{}, err
	}
	mp, err := svc.groups.Members(ctx, groupID, groupType, pm)
	if err != nil {
		return MemberPage{}, errors.Wrap(ErrFailedToRetrieveMembers, err)
	}
	return mp, nil
}

func (svc service) RemoveGroup(ctx context.Context, token, id string) error {
	if _, err := svc.Identify(ctx, token); err != nil {
		return err
	}
	return svc.groups.Delete(ctx, id)
}

func (svc service) UpdateGroup(ctx context.Context, token string, group Group) (Group, error) {
	if _, err := svc.Identify(ctx, token); err != nil {
		return Group{}, err
	}

	group.UpdatedAt = getTimestmap()
	return svc.groups.Update(ctx, group)
}

func (svc service) ViewGroup(ctx context.Context, token, id string) (Group, error) {
	if _, err := svc.Identify(ctx, token); err != nil {
		return Group{}, err
	}
	return svc.groups.RetrieveByID(ctx, id)
}

func (svc service) Assign(ctx context.Context, token string, groupID, groupType string, memberIDs ...string) error {
	if _, err := svc.Identify(ctx, token); err != nil {
		return err
	}

	if err := svc.groups.Assign(ctx, groupID, groupType, memberIDs...); err != nil {
		return err
	}

	return nil
}

func (svc service) Unassign(ctx context.Context, token string, groupID string, memberIDs ...string) error {
	if _, err := svc.Identify(ctx, token); err != nil {
		return err
	}

	return svc.groups.Unassign(ctx, groupID, memberIDs...)
}

func (svc service) ListMemberships(ctx context.Context, token string, memberID string, pm PageMetadata) (GroupPage, error) {
	if _, err := svc.Identify(ctx, token); err != nil {
		return GroupPage{}, err
	}
	return svc.groups.Memberships(ctx, memberID, pm)
}

func getTimestmap() time.Time {
	return time.Now().UTC().Round(time.Millisecond)
}

func (svc service) CreateOrg(ctx context.Context, token string, org Org) (Org, error) {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return Org{}, err
	}

	ulid, err := svc.idProvider.ID()
	if err != nil {
		return Org{}, err
	}

	timestamp := getTimestmap()

	g := Org{
		ID:          ulid,
		OwnerID:     user.ID,
		Name:        org.Name,
		Description: org.Description,
		Metadata:    org.Metadata,
		UpdatedAt:   timestamp,
		CreatedAt:   timestamp,
	}

	if err := svc.orgs.Save(ctx, g); err != nil {
		return Org{}, err
	}

	return g, nil
}

func (svc service) ListOrgs(ctx context.Context, token string, pm OrgPageMetadata) (OrgPage, error) {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return OrgPage{}, err
	}

	return svc.orgs.RetrieveAll(ctx, user.ID, pm)
}

func (svc service) ListOrgMembers(ctx context.Context, token string, orgID string, pm OrgPageMetadata) (OrgMembersPage, error) {
	if _, err := svc.Identify(ctx, token); err != nil {
		return OrgMembersPage{}, err
	}
	mp, err := svc.orgs.Members(ctx, orgID, pm)
	if err != nil {
		return OrgMembersPage{}, errors.Wrap(ErrFailedToRetrieveMembers, err)
	}
	return mp, nil
}

func (svc service) RemoveOrg(ctx context.Context, token, id string) error {
	res, err := svc.Identify(ctx, token)
	if err != nil {
		return err
	}

	return svc.orgs.Delete(ctx, res.ID, id)
}

func (svc service) UpdateOrg(ctx context.Context, token string, org Org) (Org, error) {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return Org{}, err
	}

	g := Org{
		ID:          org.ID,
		OwnerID:     user.ID,
		Name:        org.Name,
		Description: org.Description,
		UpdatedAt:   getTimestmap(),
	}

	if err := svc.orgs.Update(ctx, org); err != nil {
		return Org{}, err
	}

	return g, nil
}

func (svc service) ViewOrg(ctx context.Context, token, id string) (Org, error) {
	if _, err := svc.Identify(ctx, token); err != nil {
		return Org{}, err
	}

	return svc.orgs.RetrieveByID(ctx, id)
}

func (svc service) AssignOrg(ctx context.Context, token, orgID string, memberIDs ...string) error {
	if _, err := svc.Identify(ctx, token); err != nil {
		return err
	}

	if err := svc.orgs.Assign(ctx, orgID, memberIDs...); err != nil {
		return err
	}

	return nil
}

func (svc service) UnassignOrg(ctx context.Context, token string, orgID string, memberIDs ...string) error {
	if _, err := svc.Identify(ctx, token); err != nil {
		return err
	}

	if err := svc.orgs.Unassign(ctx, orgID, memberIDs...); err != nil {
		return err
	}

	return nil
}

func (svc service) ListOrgMemberships(ctx context.Context, token string, memberID string, pm OrgPageMetadata) (OrgPage, error) {
	if _, err := svc.Identify(ctx, token); err != nil {
		return OrgPage{}, err
	}

	return svc.orgs.Memberships(ctx, memberID, pm)
}

func (svc service) AssignOrgAccessRights(ctx context.Context, token, thingOrgID, userOrgID string) error {
	if _, err := svc.Identify(ctx, token); err != nil {
		return err
	}
	return nil
}
