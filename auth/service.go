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
	rootSubject      = "root"
	groupSubject     = "group"
	RPolicy          = "r"
	RwPolicy         = "rw"
)

var (
	// ErrFailedToRetrieveMembers failed to retrieve group members.
	ErrFailedToRetrieveMembers = errors.New("failed to retrieve org members")

	// ErrFailedToRetrieveMembership failed to retrieve memberships
	ErrFailedToRetrieveMembership = errors.New("failed to retrieve memberships")

	errIssueUser      = errors.New("failed to issue new login key")
	errIssueTmp       = errors.New("failed to issue new temporary key")
	errRevoke         = errors.New("failed to remove key")
	errRetrieve       = errors.New("failed to retrieve key data")
	errIdentify       = errors.New("failed to validate token")
	errUnknownSubject = errors.New("unknown subject")
)

type Roles interface {
	// AssignRole assigns a role to a user.
	AssignRole(ctx context.Context, id, role string) error
}

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

// AuthzReq represents an argument struct for making an authz related function calls.
type AuthzReq struct {
	Token   string
	Object  string
	Subject string
	Action  string
}

// Authz represents a authorization service. It exposes
// functionalities through `auth` to perform authorization.
type Authz interface {
	Authorize(ctx context.Context, ar AuthzReq) error
	AddPolicy(ctx context.Context, token, groupID, policy string) error
}

// Service specifies an API that must be fulfilled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
// Token is a string value of the actual Key and is used to authenticate
// an Auth service request.
type Service interface {
	Authn
	Authz
	Roles
	Orgs
}

var _ Service = (*service)(nil)

type service struct {
	orgs          OrgRepository
	users         mainflux.UsersServiceClient
	things        mainflux.ThingsServiceClient
	keys          KeyRepository
	roles         RolesRepository
	idProvider    mainflux.IDProvider
	tokenizer     Tokenizer
	loginDuration time.Duration
}

// New instantiates the auth service implementation.
func New(orgs OrgRepository, tc mainflux.ThingsServiceClient, uc mainflux.UsersServiceClient, keys KeyRepository, roles RolesRepository, idp mainflux.IDProvider, tokenizer Tokenizer, duration time.Duration) Service {
	return &service{
		tokenizer:     tokenizer,
		things:        tc,
		orgs:          orgs,
		users:         uc,
		keys:          keys,
		roles:         roles,
		idProvider:    idp,
		loginDuration: duration,
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
	return svc.identify(ctx, token)
}

func (svc service) Authorize(ctx context.Context, ar AuthzReq) error {
	user, err := svc.identify(ctx, ar.Token)
	if err != nil {
		return err
	}

	switch ar.Subject {
	case rootSubject:
		return svc.canAccessRoot(ctx, user.ID)
	case groupSubject:
		return svc.canAccessGroup(ctx, user.ID, ar.Object, ar.Action)
	default:
		return errUnknownSubject
	}
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

func (svc service) CreateOrg(ctx context.Context, token string, o Org) (Org, error) {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return Org{}, err
	}

	id, err := svc.idProvider.ID()
	if err != nil {
		return Org{}, err
	}

	timestamp := getTimestmap()

	org := Org{
		ID:          id,
		OwnerID:     user.ID,
		Name:        o.Name,
		Description: o.Description,
		Metadata:    o.Metadata,
		UpdatedAt:   timestamp,
		CreatedAt:   timestamp,
	}

	if err := svc.orgs.Save(ctx, org); err != nil {
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

	return org, nil
}

func (svc service) ListOrgs(ctx context.Context, token string, admin bool, pm PageMetadata) (OrgsPage, error) {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return OrgsPage{}, err
	}

	if admin {
		if err := svc.canAccessRoot(ctx, user.ID); err == nil {
			return svc.orgs.RetrieveByAdmin(ctx, pm)
		}
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

	var groupIDs []string
	for _, g := range gPage.GroupRelations {
		groupIDs = append(groupIDs, g.GroupID)
	}

	if err := svc.orgs.UnassignGroups(ctx, id, groupIDs...); err != nil {
		return err
	}

	return svc.orgs.Delete(ctx, user.ID, id)
}

func (svc service) UpdateOrg(ctx context.Context, token string, o Org) (Org, error) {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return Org{}, err
	}

	if err := svc.canEditOrg(ctx, o.ID, user.ID); err != nil {
		return Org{}, err
	}

	org := Org{
		ID:          o.ID,
		OwnerID:     user.ID,
		Name:        o.Name,
		Description: o.Description,
		Metadata:    o.Metadata,
		UpdatedAt:   getTimestmap(),
	}

	if err := svc.orgs.Update(ctx, org); err != nil {
		return Org{}, err
	}

	return org, nil
}

func (svc service) ViewOrg(ctx context.Context, token, id string) (Org, error) {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return Org{}, err
	}

	if err := svc.canAccessOrg(ctx, id, user); err != nil {
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
	var mrs []MemberRelation
	for _, m := range mbs {
		mr := MemberRelation{
			OrgID:     orgID,
			MemberID:  m.ID,
			Role:      m.Role,
			UpdatedAt: timestamp,
			CreatedAt: timestamp,
		}

		mrs = append(mrs, mr)
	}

	if err := svc.orgs.AssignMembers(ctx, mrs...); err != nil {
		return err
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

	var mrs []MemberRelation
	for _, m := range mbs {
		mr := MemberRelation{
			OrgID:     orgID,
			MemberID:  m.ID,
			Role:      m.Role,
			UpdatedAt: getTimestmap(),
		}

		mrs = append(mrs, mr)
	}

	if err := svc.orgs.UpdateMembers(ctx, mrs...); err != nil {
		return err
	}

	return nil
}

func (svc service) ListOrgMembers(ctx context.Context, token string, orgID string, pm PageMetadata) (MembersPage, error) {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return MembersPage{}, err
	}

	if err := svc.canAccessOrg(ctx, orgID, user); err != nil {
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
	var grs []GroupRelation
	for _, groupID := range groupIDs {
		gr := GroupRelation{
			OrgID:     orgID,
			GroupID:   groupID,
			CreatedAt: timestamp,
			UpdatedAt: timestamp,
		}

		grs = append(grs, gr)
	}

	mp, err := svc.ListOrgMembers(ctx, token, orgID, PageMetadata{})
	if err != nil {
		return err
	}

	for _, member := range mp.Members {
		if err := svc.orgs.SavePolicy(ctx, member.ID, RwPolicy, groupIDs...); err != nil {
			return err
		}
	}

	if err := svc.orgs.AssignGroups(ctx, grs...); err != nil {
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

	if err := svc.canAccessOrg(ctx, orgID, user); err != nil {
		return GroupsPage{}, err
	}

	mp, err := svc.orgs.RetrieveGroups(ctx, orgID, pm)
	if err != nil {
		return GroupsPage{}, errors.Wrap(ErrFailedToRetrieveMembers, err)
	}

	var groupIDs []string
	for _, g := range mp.GroupRelations {
		groupIDs = append(groupIDs, g.GroupID)
	}

	var groups []Group
	if len(groupIDs) > 0 {
		greq := mainflux.GroupsReq{Ids: groupIDs}
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

	if err := svc.canAccessRoot(ctx, user.ID); err == nil {
		return svc.orgs.RetrieveMemberships(ctx, memberID, pm)
	}

	if user.ID != memberID {
		return OrgsPage{}, errors.ErrAuthorization
	}

	return svc.orgs.RetrieveMemberships(ctx, memberID, pm)
}

func (svc service) canAccessGroup(ctx context.Context, userID, Object, action string) error {
	gp := GroupsPolicy{
		MemberID: userID,
		GroupID:  Object,
	}

	policy, err := svc.orgs.RetrievePolicy(ctx, gp)
	if err != nil {
		return err
	}

	if policy != action && policy != RwPolicy {
		return errors.ErrAuthorization
	}

	return nil
}

func (svc service) AddPolicy(ctx context.Context, token, groupID, policy string) error {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return err
	}

	if err := svc.orgs.SavePolicy(ctx, user.ID, policy, groupID); err != nil {
		return err
	}

	return nil
}

func (svc service) Backup(ctx context.Context, token string) (Backup, error) {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return Backup{}, err
	}

	if err := svc.canAccessRoot(ctx, user.ID); err != nil {
		return Backup{}, err
	}

	orgs, err := svc.orgs.RetrieveAll(ctx)
	if err != nil {
		return Backup{}, err
	}

	mrs, err := svc.orgs.RetrieveAllMemberRelations(ctx)
	if err != nil {
		return Backup{}, err
	}

	grs, err := svc.orgs.RetrieveAllGroupRelations(ctx)
	if err != nil {
		return Backup{}, err
	}

	backup := Backup{
		Orgs:            orgs,
		MemberRelations: mrs,
		GroupRelations:  grs,
	}

	return backup, nil
}

func (svc service) Restore(ctx context.Context, token string, backup Backup) error {
	user, err := svc.Identify(ctx, token)
	if err != nil {
		return err
	}

	if err := svc.canAccessRoot(ctx, user.ID); err != nil {
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

func (svc service) AssignRole(ctx context.Context, id, role string) error {
	if err := svc.roles.SaveRole(ctx, id, role); err != nil {
		return err
	}

	return nil
}

func (svc service) canAccessRoot(ctx context.Context, id string) error {
	role, err := svc.roles.RetrieveRole(ctx, id)
	if err != nil {
		return err
	}

	if role != RoleAdmin && role != RoleRootAdmin {
		return errors.ErrAuthorization
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

func (svc service) canAccessOrg(ctx context.Context, orgID string, user Identity) error {
	if err := svc.canAccessRoot(ctx, user.ID); err == nil {
		return nil
	}

	role, err := svc.orgs.RetrieveRole(ctx, user.ID, orgID)
	if err != nil {
		return err
	}

	if role != OwnerRole && role != AdminRole && role != EditorRole && role != ViewerRole {
		return errors.ErrAuthorization
	}

	return nil
}

func (svc service) identify(ctx context.Context, token string) (Identity, error) {
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
