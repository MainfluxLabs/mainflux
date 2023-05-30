// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

const (
	recoveryDuration = 5 * time.Minute
	ViewerRole       = "viewer"
	AdminRole        = "admin"
	OwnerRole        = "owner"
	EditorRole       = "editor"
)

var (
	// ErrFailedToRetrieveMembers failed to retrieve group members.
	ErrFailedToRetrieveMembers = errors.New("failed to retrieve org members")

	// ErrFailedToRetrieveMembership failed to retrieve memberships
	ErrFailedToRetrieveMembership = errors.New("failed to retrieve memberships")

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

// AuthReq represents an argument struct for making an authz related
// function calls.
type AuthzReq struct {
	Email string
}

// Authz represents a authorization service. It exposes
// functionalities through `auth` to perform authorization.
type Authz interface {
	// Authorize indicates if user is admin.
	Authorize(ctx context.Context, pr AuthzReq) error

	// CanAccessGroup indicates if user can access group for a given token.
	CanAccessGroup(ctx context.Context, token, groupID string) error
}

// Service specifies an API that must be fulfilled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
// Token is a string value of the actual Key and is used to authenticate
// an Auth service request.
type Service interface {
	Authn
	Authz

	// OrgService implements orgs API, creating orgs, assigning members and groups
	OrgService
}

var _ Service = (*service)(nil)

type service struct {
	orgs          OrgRepository
	users         mainflux.UsersServiceClient
	things        mainflux.ThingsServiceClient
	keys          KeyRepository
	idProvider    mainflux.IDProvider
	tokenizer     Tokenizer
	loginDuration time.Duration
	adminEmail    string
}

// New instantiates the auth service implementation.
func New(orgs OrgRepository, tc mainflux.ThingsServiceClient, uc mainflux.UsersServiceClient, keys KeyRepository, idp mainflux.IDProvider, tokenizer Tokenizer, duration time.Duration, adminEmail string) Service {
	return &service{
		tokenizer:     tokenizer,
		things:        tc,
		orgs:          orgs,
		users:         uc,
		keys:          keys,
		idProvider:    idp,
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

func (svc service) Authorize(ctx context.Context, pr AuthzReq) error {
	if pr.Email != svc.adminEmail {
		return errors.ErrAuthorization
	}

	return nil
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

func getTimestmap() time.Time {
	return time.Now().UTC().Round(time.Millisecond)
}

func (svc service) CreateOrg(ctx context.Context, token string, org Org) (Org, error) {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return Org{}, err
	}

	id, err := svc.idProvider.ID()
	if err != nil {
		return Org{}, err
	}

	timestamp := getTimestmap()

	g := Org{
		ID:          id,
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

	mr := MemberRelation{
		OrgID:     id,
		MemberID:  user.ID,
		Role:      OwnerRole,
		CreatedAt: timestamp,
		UpdatedAt: timestamp,
	}

	if err := svc.orgs.AssignMembers(ctx, mr); err != nil {
		return Org{}, err
	}

	return g, nil
}

func (svc service) ListOrgs(ctx context.Context, token string, pm PageMetadata) (OrgsPage, error) {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return OrgsPage{}, err
	}

	return svc.orgs.RetrieveByOwner(ctx, user.ID, pm)
}

func (svc service) RemoveOrg(ctx context.Context, token, id string) error {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return err
	}

	if err := svc.isOwner(ctx, id, user.ID); err != nil {
		return err
	}

	mPage, err := svc.orgs.RetrieveMembers(ctx, id, PageMetadata{})
	if err != nil {
		return err
	}

	var memberIDs []string
	for _, m := range mPage.Members {
		memberIDs = append(memberIDs, m.ID)
	}

	if err := svc.orgs.UnassignMembers(ctx, id, memberIDs...); err != nil {
		return err
	}

	gPage, err := svc.orgs.RetrieveGroups(ctx, id, PageMetadata{})
	if err != nil {
		return err
	}

	if err := svc.orgs.UnassignGroups(ctx, id, gPage.GroupIDs...); err != nil {
		return err
	}

	return svc.orgs.Delete(ctx, user.ID, id)
}

func (svc service) UpdateOrg(ctx context.Context, token string, org Org) (Org, error) {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return Org{}, err
	}

	if err := svc.canEditOrg(ctx, org.ID, user.ID); err != nil {
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
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return Org{}, err
	}

	if err := svc.canAccessOrg(ctx, id, user.ID); err != nil {
		return Org{}, err
	}

	org, err := svc.orgs.RetrieveByID(ctx, id)
	if err != nil {
		return Org{}, err
	}

	return org, nil
}

func (svc service) AssignMembersByIDs(ctx context.Context, token, orgID string, memberIDs ...string) error {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return err
	}

	if err := svc.canEditMembers(ctx, orgID, user.ID, memberIDs...); err != nil {
		return err
	}

	if err := svc.orgs.AssignMembers(ctx, MemberRelation{}); err != nil {
		return err
	}

	return nil
}

func (svc service) AssignMembers(ctx context.Context, token, orgID string, members ...Member) error {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return err
	}

	if err := svc.canEditOrg(ctx, orgID, user.ID); err != nil {
		return err
	}

	var memberEmails []string
	var member = make(map[string]string)
	for _, m := range members {
		member[m.Email] = m.Role
		memberEmails = append(memberEmails, m.Email)
	}

	muReq := mainflux.UsersByEmailsReq{Emails: memberEmails}
	usr, err := svc.users.GetUsersByEmails(ctx, &muReq)
	if err != nil {
		return err
	}

	mbs := []Member{}
	for _, user := range usr.Users {
		mbs = append(mbs, Member{
			Role: member[user.Email],
			ID:   user.Id,
		})

	}

	timestamp := getTimestmap()
	for _, m := range mbs {
		mr := MemberRelation{
			OrgID:     orgID,
			MemberID:  m.ID,
			Role:      m.Role,
			UpdatedAt: timestamp,
			CreatedAt: timestamp,
		}

		if err := svc.orgs.AssignMembers(ctx, mr); err != nil {
			return err
		}
	}

	return nil
}

func (svc service) UnassignMembersByIDs(ctx context.Context, token string, orgID string, memberIDs ...string) error {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return err
	}

	if err := svc.canEditMembers(ctx, orgID, user.ID, memberIDs...); err != nil {
		return err
	}

	if err := svc.orgs.UnassignMembers(ctx, orgID, memberIDs...); err != nil {
		return err
	}

	return nil
}

func (svc service) UnassignMembers(ctx context.Context, token string, orgID string, memberIDs ...string) error {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return err
	}

	if err := svc.canEditMembers(ctx, orgID, user.ID, memberIDs...); err != nil {
		return err
	}

	if err := svc.orgs.UnassignMembers(ctx, orgID, memberIDs...); err != nil {
		return err
	}

	return nil
}

func (svc service) UpdateMembers(ctx context.Context, token, orgID string, members ...Member) error {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return err
	}

	if err := svc.canEditOrg(ctx, orgID, user.ID); err != nil {
		return err
	}

	var memberEmails []string
	var member = make(map[string]string)
	for _, m := range members {
		member[m.Email] = m.Role
		memberEmails = append(memberEmails, m.Email)
	}

	muReq := mainflux.UsersByEmailsReq{Emails: memberEmails}
	usr, err := svc.users.GetUsersByEmails(ctx, &muReq)
	if err != nil {
		return err
	}

	mbs := []Member{}
	for _, user := range usr.Users {
		mbs = append(mbs, Member{
			Role: member[user.Email],
			ID:   user.Id,
		})

	}

	var memberRelations []MemberRelation
	for _, m := range mbs {
		mr := MemberRelation{
			OrgID:     orgID,
			MemberID:  m.ID,
			Role:      m.Role,
			UpdatedAt: getTimestmap(),
		}

		memberRelations = append(memberRelations, mr)
	}

	if err := svc.orgs.UpdateMembers(ctx, memberRelations...); err != nil {
		return err
	}

	return nil
}

func (svc service) ListOrgMembers(ctx context.Context, token string, orgID string, pm PageMetadata) (MembersPage, error) {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return MembersPage{}, err
	}

	if err := svc.canAccessOrg(ctx, orgID, user.ID); err != nil {
		return MembersPage{}, err
	}

	mp, err := svc.orgs.RetrieveMembers(ctx, orgID, pm)
	if err != nil {
		return MembersPage{}, errors.Wrap(ErrFailedToRetrieveMembers, err)
	}

	var memberIDs []string
	for _, m := range mp.Members {
		memberIDs = append(memberIDs, m.ID)
	}
	var members []Member
	if len(mp.Members) > 0 {
		usrReq := mainflux.UsersByIDsReq{Ids: memberIDs}
		page, err := svc.users.GetUsersByIDs(ctx, &usrReq)
		if err != nil {
			return MembersPage{}, err
		}

		emails := make(map[string]string)
		for _, user := range page.Users {
			emails[user.Id] = user.GetEmail()
		}

		for _, m := range mp.Members {
			email, ok := emails[m.ID]
			if !ok {
				return MembersPage{}, err
			}

			mbr := Member{
				ID:    m.ID,
				Email: email,
				Role:  m.Role,
			}
			members = append(members, mbr)
		}
	}

	mpg := MembersPage{
		Members: members,
		PageMetadata: PageMetadata{
			Total:  mp.Total,
			Offset: mp.Offset,
			Limit:  mp.Limit,
		},
	}

	return mpg, nil
}

func (svc service) AssignGroups(ctx context.Context, token, orgID string, groupIDs ...string) error {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return err
	}

	if err := svc.canEditGroups(ctx, orgID, user.ID); err != nil {
		return err
	}

	timestamp := getTimestmap()
	var gr []GroupRelation
	for _, groupID := range groupIDs {
		g := GroupRelation{
			OrgID:     orgID,
			GroupID:   groupID,
			CreatedAt: timestamp,
			UpdatedAt: timestamp,
		}

		gr = append(gr, g)
	}

	if err := svc.orgs.AssignGroups(ctx, gr...); err != nil {
		return err
	}

	return nil
}

func (svc service) UnassignGroups(ctx context.Context, token string, orgID string, groupIDs ...string) error {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return err
	}

	if err := svc.canEditGroups(ctx, orgID, user.ID); err != nil {
		return err
	}

	if err := svc.orgs.UnassignGroups(ctx, orgID, groupIDs...); err != nil {
		return err
	}

	return nil
}

func (svc service) ListOrgGroups(ctx context.Context, token string, orgID string, pm PageMetadata) (GroupsPage, error) {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return GroupsPage{}, err
	}

	if err := svc.canAccessOrg(ctx, orgID, user.ID); err != nil {
		return GroupsPage{}, err
	}

	mp, err := svc.orgs.RetrieveGroups(ctx, orgID, pm)
	if err != nil {
		return GroupsPage{}, errors.Wrap(ErrFailedToRetrieveMembers, err)
	}

	var groups []Group
	if len(mp.GroupIDs) > 0 {
		greq := mainflux.GroupsReq{Ids: mp.GroupIDs}
		resp, err := svc.things.GetGroupsByIDs(ctx, &greq)
		if err != nil {
			return GroupsPage{}, err
		}

		for _, g := range resp.Groups {
			gr := Group{
				ID:          g.Id,
				OwnerID:     g.OwnerID,
				Name:        g.Name,
				Description: g.Description,
			}
			groups = append(groups, gr)
		}
	}

	pg := GroupsPage{
		Groups: groups,
		PageMetadata: PageMetadata{
			Total:  mp.Total,
			Offset: mp.Offset,
			Limit:  mp.Limit,
		},
	}

	return pg, nil
}

func (svc service) ListOrgMemberships(ctx context.Context, token string, memberID string, pm PageMetadata) (OrgsPage, error) {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return OrgsPage{}, err
	}

	if user.ID != memberID {
		return OrgsPage{}, errors.ErrAuthorization
	}

	return svc.orgs.RetrieveMemberships(ctx, memberID, pm)
}

func (svc service) CanAccessGroup(ctx context.Context, token, groupID string) error {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return err
	}

	// Retrieve orgs where group is assigned
	op, err := svc.orgs.RetrieveByGroupID(ctx, groupID)
	if err != nil {
		return err
	}

	for _, org := range op.Orgs {
		if err := svc.canAccessOrg(ctx, org.ID, user.ID); err == nil {
			return nil
		}
	}

	return errors.ErrAuthorization
}

func (svc service) Backup(ctx context.Context, token string) (Backup, error) {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return Backup{}, err
	}

	pr := AuthzReq{Email: user.Email}
	if err := svc.Authorize(ctx, pr); err != nil {
		return Backup{}, err
	}

	orgs, err := svc.orgs.RetrieveAll(ctx)
	if err != nil {
		return Backup{}, err
	}

	members, err := svc.orgs.RetrieveAllMemberRelations(ctx)
	if err != nil {
		return Backup{}, err
	}

	groups, err := svc.orgs.RetrieveAllGroupRelations(ctx)
	if err != nil {
		return Backup{}, err
	}

	backup := Backup{
		Orgs:            orgs,
		MemberRelations: members,
		GroupRelations:  groups,
	}

	return backup, nil
}

func (svc service) Restore(ctx context.Context, token string, backup Backup) error {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return err
	}

	pr := AuthzReq{Email: user.Email}
	if err := svc.Authorize(ctx, pr); err != nil {
		return err
	}

	if err := svc.orgs.Save(ctx, backup.Orgs...); err != nil {
		return err
	}

	if err := svc.orgs.AssignMembers(ctx, backup.MemberRelations...); err != nil {
		return err
	}

	if err := svc.orgs.AssignGroups(ctx, backup.GroupRelations...); err != nil {
		return err
	}

	return nil
}

func (svc service) isOwner(ctx context.Context, orgID, userID string) error {
	role, err := svc.orgs.RetrieveRole(ctx, userID, orgID)
	if err != nil {
		return err
	}

	if role != OwnerRole {
		return errors.ErrAuthorization
	}

	return nil
}

func (svc service) canEditOrg(ctx context.Context, orgID, userID string) error {
	role, err := svc.orgs.RetrieveRole(ctx, userID, orgID)
	if err != nil {
		return err
	}

	if role != OwnerRole && role != AdminRole {
		return errors.ErrAuthorization
	}

	return nil
}

func (svc service) canEditMembers(ctx context.Context, orgID, userID string, memberIDs ...string) error {
	if err := svc.canEditOrg(ctx, orgID, userID); err != nil {
		return err
	}

	for _, memberID := range memberIDs {
		role, err := svc.orgs.RetrieveRole(ctx, memberID, orgID)
		if err != nil {
			return err
		}

		if role == OwnerRole {
			return errors.ErrAuthorization
		}
	}

	return nil
}

func (svc service) canEditGroups(ctx context.Context, orgID, userID string) error {
	role, err := svc.orgs.RetrieveRole(ctx, userID, orgID)
	if err != nil {
		return err
	}

	if role != OwnerRole && role != AdminRole && role != EditorRole {
		return errors.ErrAuthorization
	}

	return nil
}

func (svc service) canAccessOrg(ctx context.Context, orgID, userID string) error {
	role, err := svc.orgs.RetrieveRole(ctx, userID, orgID)
	if err != nil {
		return err
	}

	if role != OwnerRole && role != AdminRole && role != EditorRole && role != ViewerRole {
		return errors.ErrAuthorization
	}

	return nil
}
