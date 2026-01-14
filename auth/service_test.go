// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package auth_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/auth/jwt"
	"github.com/MainfluxLabs/mainflux/auth/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	thmocks "github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	redirectPathInvite = "/view-invite"

	secret            = "secret"
	email             = "test@example.com"
	superAdminEmail   = "admin@example.com"
	ownerEmail        = "owner@test.com"
	viewerEmail       = "viewer@test.com"
	editorEmail       = "editor@test.com"
	adminEmail        = "admin@test.com"
	unregisteredEmail = "unregistered@test.com"
	id                = "testID"
	ownerID           = "ownerID"
	adminID           = "adminID"
	editorID          = "editorID"
	viewerID          = "viewerID"
	rootAdminID       = "rootAdminID"
	description       = "description"
	name              = "name"
	invalid           = "invalid"
	n                 = 10

	loginDuration  = 30 * time.Minute
	inviteDuration = 7 * 24 * time.Hour
)

var (
	org           = auth.Org{Name: name, Description: description}
	memberships   = []auth.OrgMembership{{MemberID: "1", Email: adminEmail, Role: auth.Admin}, {MemberID: "2", Email: editorEmail, Role: auth.Editor}, {MemberID: "3", Email: viewerEmail, Role: auth.Viewer}}
	usersByEmails = map[string]users.User{adminEmail: {ID: adminID, Email: adminEmail}, editorEmail: {ID: editorID, Email: editorEmail}, viewerEmail: {ID: viewerID, Email: viewerEmail}, ownerEmail: {ID: ownerID, Email: ownerEmail}}
	usersByIDs    = map[string]users.User{adminID: {ID: adminID, Email: adminEmail}, editorID: {ID: editorID, Email: editorEmail}, viewerID: {ID: viewerID, Email: viewerEmail}, ownerID: {ID: ownerID, Email: ownerEmail}}
	idProvider    = uuid.New()
)

func newService() auth.Service {
	keyRepo := mocks.NewKeyRepository()
	idMockProvider := uuid.NewMock()
	membsRepo := mocks.NewOrgMembershipsRepository()
	orgRepo := mocks.NewOrgRepository(membsRepo)
	roleRepo := mocks.NewRolesRepository()
	invitesRepo := mocks.NewInvitesRepository()
	emailerMock := mocks.NewEmailer()

	for i := 1; i <= 10; i++ {
		uEmail := fmt.Sprintf("example%d@test.com", i)
		uID := fmt.Sprintf("example%d", i)

		user := users.User{
			ID:    uID,
			Email: uEmail,
		}

		usersByEmails[uEmail] = user
		usersByIDs[uID] = user
	}

	uc := mocks.NewUsersService(usersByIDs, usersByEmails)
	tc := thmocks.NewThingsServiceClient(nil, nil, createGroups())
	t := jwt.New(secret)

	return auth.New(orgRepo, tc, uc, keyRepo, roleRepo, membsRepo, invitesRepo, emailerMock, idMockProvider, t, loginDuration, inviteDuration)
}

func createGroups() map[string]things.Group {
	groups := make(map[string]things.Group, n)
	for i := 0; i < n; i++ {
		groupId := fmt.Sprintf(id+"-%d", i)
		groups[groupId] = things.Group{
			ID:          groupId,
			Name:        fmt.Sprintf(name+"-%d", i),
			Description: fmt.Sprintf(description+"-%d", i),
			Metadata:    map[string]any{"meta": "data"},
		}
	}

	return groups
}

func TestIssue(t *testing.T) {
	svc := newService()
	_, secret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	cases := []struct {
		desc  string
		key   auth.Key
		token string
		err   error
	}{
		{
			desc: "issue login key",
			key: auth.Key{
				Type:     auth.LoginKey,
				IssuedAt: time.Now(),
			},
			token: secret,
			err:   nil,
		},
		{
			desc: "issue login key with no time",
			key: auth.Key{
				Type: auth.LoginKey,
			},
			token: secret,
			err:   auth.ErrInvalidKeyIssuedAt,
		},
		{
			desc: "issue API key",
			key: auth.Key{
				Type:     auth.APIKey,
				IssuedAt: time.Now(),
			},
			token: secret,
			err:   nil,
		},
		{
			desc: "issue API key with an invalid token",
			key: auth.Key{
				Type:     auth.APIKey,
				IssuedAt: time.Now(),
			},
			token: "invalid",
			err:   errors.ErrAuthentication,
		},
		{
			desc: "issue API key with no time",
			key: auth.Key{
				Type: auth.APIKey,
			},
			token: secret,
			err:   auth.ErrInvalidKeyIssuedAt,
		},
		{
			desc: "issue recovery key",
			key: auth.Key{
				Type:     auth.RecoveryKey,
				IssuedAt: time.Now(),
			},
			token: "",
			err:   nil,
		},
		{
			desc: "issue recovery with no issue time",
			key: auth.Key{
				Type: auth.RecoveryKey,
			},
			token: secret,
			err:   auth.ErrInvalidKeyIssuedAt,
		},
	}

	for _, tc := range cases {
		_, _, err := svc.Issue(context.Background(), tc.token, tc.key)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRevoke(t *testing.T) {
	svc := newService()
	_, secret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	key := auth.Key{
		Type:     auth.APIKey,
		IssuedAt: time.Now(),
		IssuerID: id,
		Subject:  email,
	}
	newKey, _, err := svc.Issue(context.Background(), secret, key)
	assert.Nil(t, err, fmt.Sprintf("Issuing user's key expected to succeed: %s", err))

	cases := []struct {
		desc  string
		id    string
		token string
		err   error
	}{
		{
			desc:  "revoke login key",
			id:    newKey.ID,
			token: secret,
			err:   nil,
		},
		{
			desc:  "revoke non-existing login key",
			id:    newKey.ID,
			token: secret,
			err:   nil,
		},
		{
			desc:  "revoke with empty login key",
			id:    newKey.ID,
			token: "",
			err:   errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		err := svc.Revoke(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieve(t *testing.T) {
	svc := newService()
	_, secret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), Subject: email, IssuerID: id})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	key := auth.Key{
		ID:       "id",
		Type:     auth.APIKey,
		IssuerID: id,
		Subject:  email,
		IssuedAt: time.Now(),
	}

	_, userToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	apiKey, apiToken, err := svc.Issue(context.Background(), secret, key)
	assert.Nil(t, err, fmt.Sprintf("Issuing login's key expected to succeed: %s", err))

	_, resetToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.RecoveryKey, IssuedAt: time.Now()})
	assert.Nil(t, err, fmt.Sprintf("Issuing reset key expected to succeed: %s", err))

	cases := []struct {
		desc  string
		id    string
		token string
		err   error
	}{
		{
			desc:  "retrieve login key",
			id:    apiKey.ID,
			token: userToken,
			err:   nil,
		},
		{
			desc:  "retrieve non-existing login key",
			id:    "invalid",
			token: userToken,
			err:   dbutil.ErrNotFound,
		},
		{
			desc:  "retrieve with wrong login key",
			id:    apiKey.ID,
			token: "wrong",
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "retrieve with API token",
			id:    apiKey.ID,
			token: apiToken,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "retrieve with reset token",
			id:    apiKey.ID,
			token: resetToken,
			err:   errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		_, err := svc.RetrieveKey(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestIdentify(t *testing.T) {
	svc := newService()

	_, loginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	_, recoverySecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.RecoveryKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing reset key expected to succeed: %s", err))

	_, apiSecret, err := svc.Issue(context.Background(), loginSecret, auth.Key{Type: auth.APIKey, IssuerID: id, Subject: email, IssuedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute)})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	exp1 := time.Now().Add(-2 * time.Second)
	_, expSecret, err := svc.Issue(context.Background(), loginSecret, auth.Key{Type: auth.APIKey, IssuedAt: time.Now(), ExpiresAt: exp1})
	assert.Nil(t, err, fmt.Sprintf("Issuing expired login key expected to succeed: %s", err))

	_, invalidSecret, err := svc.Issue(context.Background(), loginSecret, auth.Key{Type: 22, IssuedAt: time.Now()})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	cases := []struct {
		desc string
		key  string
		idt  auth.Identity
		err  error
	}{
		{
			desc: "identify login key",
			key:  loginSecret,
			idt:  auth.Identity{id, email},
			err:  nil,
		},
		{
			desc: "identify recovery key",
			key:  recoverySecret,
			idt:  auth.Identity{id, email},
			err:  nil,
		},
		{
			desc: "identify API key",
			key:  apiSecret,
			idt:  auth.Identity{id, email},
			err:  nil,
		},
		{
			desc: "identify expired API key",
			key:  expSecret,
			idt:  auth.Identity{},
			err:  auth.ErrAPIKeyExpired,
		},
		{
			desc: "identify expired key",
			key:  invalidSecret,
			idt:  auth.Identity{},
			err:  errors.ErrAuthentication,
		},
		{
			desc: "identify invalid key",
			key:  "invalid",
			idt:  auth.Identity{},
			err:  errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		idt, err := svc.Identify(context.Background(), tc.key)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.idt, idt, fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.idt, idt))
	}
}

func TestAuthorize(t *testing.T) {
	svc := newService()

	_, adminToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: adminID, Subject: adminEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	err = svc.AssignRole(context.Background(), adminID, auth.RoleAdmin)
	require.Nil(t, err, fmt.Sprintf("saving role expected to succeed: %s", err))

	pr := auth.AuthzReq{Token: adminToken, Subject: auth.RootSub}
	err = svc.Authorize(context.Background(), pr)
	require.Nil(t, err, fmt.Sprintf("authorizing initial %v authz request expected to succeed: %s", pr, err))
}

func TestCreateOrg(t *testing.T) {
	svc := newService()

	_, ownerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: ownerID, Subject: ownerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	cases := []struct {
		desc  string
		token string
		org   auth.Org
		err   error
	}{
		{
			desc:  "create org",
			token: ownerToken,
			org:   org,
			err:   nil,
		}, {
			desc:  "create org with wrong credentials",
			token: "invalid",
			org:   org,
			err:   errors.ErrAuthentication,
		}, {
			desc:  "create org without credentials",
			token: "",
			org:   org,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "create org without name and description",
			token: ownerToken,
			org:   auth.Org{},
			err:   nil,
		},
	}

	for _, tc := range cases {
		_, err := svc.CreateOrg(context.Background(), tc.token, tc.org)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}

}

func TestListOrgs(t *testing.T) {
	svc := newService()

	_, ownerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: ownerID, Subject: ownerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, superAdminToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: rootAdminID, Subject: superAdminEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	for i := 0; i < n; i++ {
		org.Name = fmt.Sprintf("org-%d", i)
		org.Description = fmt.Sprintf("description-%d", i)
		_, err := svc.CreateOrg(context.Background(), ownerToken, org)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	}

	err = svc.AssignRole(context.Background(), rootAdminID, auth.RoleRootAdmin)
	require.Nil(t, err, fmt.Sprintf("saving role expected to succeed: %s", err))

	cases := []struct {
		desc  string
		token string
		meta  apiutil.PageMetadata
		size  uint64
		err   error
	}{
		{
			desc:  "list orgs",
			token: ownerToken,
			meta: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		}, {
			desc:  "list orgs as system admin",
			token: superAdminToken,
			meta: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		}, {
			desc:  "list orgs with wrong credentials",
			token: invalid,
			meta:  apiutil.PageMetadata{},
			size:  0,
			err:   errors.ErrAuthentication,
		}, {
			desc:  "list orgs without credentials",
			token: "",
			meta:  apiutil.PageMetadata{},
			size:  0,
			err:   errors.ErrAuthentication,
		}, {
			desc:  "list half of total orgs",
			token: ownerToken,
			meta: apiutil.PageMetadata{
				Offset: n / 2,
				Limit:  n,
			},
			size: n / 2,
			err:  nil,
		}, {
			desc:  "list last org",
			token: ownerToken,
			meta: apiutil.PageMetadata{
				Offset: n - 1,
				Limit:  n,
			},
			size: 1,
			err:  nil,
		},
	}

	for _, tc := range cases {
		page, err := svc.ListOrgs(context.Background(), tc.token, tc.meta)
		size := uint64(len(page.Orgs))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %d got %d\n", tc.desc, tc.size, size))
	}
}

func TestRemoveOrg(t *testing.T) {
	svc := newService()

	_, ownerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: ownerID, Subject: ownerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, viewerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: viewerID, Subject: viewerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, editorToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: editorID, Subject: editorEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, adminToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: adminID, Subject: adminEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	res, err := svc.CreateOrg(context.Background(), ownerToken, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	err = svc.CreateOrgMemberships(context.Background(), ownerToken, res.ID, memberships...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := []struct {
		desc  string
		token string
		id    string
		err   error
	}{
		{
			desc:  "remove org with wrong credentials",
			token: invalid,
			id:    res.ID,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "remove org without credentials",
			token: "",
			id:    res.ID,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "remove non-existing org",
			token: ownerToken,
			id:    invalid,
			err:   dbutil.ErrNotFound,
		},
		{
			desc:  "remove org as viewer",
			token: viewerToken,
			id:    res.ID,
			err:   errors.ErrAuthorization,
		},
		{
			desc:  "remove org as editor",
			token: editorToken,
			id:    res.ID,
			err:   errors.ErrAuthorization,
		},
		{
			desc:  "remove org as admin",
			token: adminToken,
			id:    res.ID,
			err:   errors.ErrAuthorization,
		},
		{
			desc:  "remove org as owner",
			token: ownerToken,
			id:    res.ID,
			err:   nil,
		},
		{
			desc:  "remove removed org",
			token: ownerToken,
			id:    res.ID,
			err:   dbutil.ErrNotFound,
		},
		{
			desc:  "remove non-existing org",
			token: ownerToken,
			id:    invalid,
			err:   dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveOrgs(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateOrg(t *testing.T) {
	svc := newService()

	_, ownerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: ownerID, Subject: ownerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, viewerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: viewerID, Subject: viewerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, editorToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: editorID, Subject: editorEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, adminToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: adminID, Subject: adminEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	res, err := svc.CreateOrg(context.Background(), ownerToken, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	err = svc.CreateOrgMemberships(context.Background(), ownerToken, res.ID, memberships...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	upOrg := auth.Org{
		ID:          res.ID,
		Name:        "updated_name",
		Description: "updated_description",
	}

	cases := []struct {
		desc  string
		token string
		org   auth.Org
		err   error
	}{
		{
			desc:  "update org as viewer",
			token: viewerToken,
			org:   upOrg,
			err:   errors.ErrAuthorization,
		},
		{
			desc:  "update org as editor",
			token: editorToken,
			org:   upOrg,
			err:   errors.ErrAuthorization,
		},
		{
			desc:  "update org as admin",
			token: adminToken,
			org:   upOrg,
			err:   nil,
		},
		{
			desc:  "update org as owner",
			token: ownerToken,
			org:   upOrg,
			err:   nil,
		},
		{
			desc:  "update org with wrong credentials",
			token: invalid,
			org:   upOrg,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "update org without credentials",
			token: "",
			org:   upOrg,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "update non-existing org",
			token: ownerToken,
			org:   auth.Org{},
			err:   dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		_, err := svc.UpdateOrg(context.Background(), tc.token, tc.org)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestViewOrg(t *testing.T) {
	svc := newService()

	_, ownerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: ownerID, Subject: ownerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, viewerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: viewerID, Subject: viewerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, editorToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: editorID, Subject: editorEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, adminToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: adminID, Subject: adminEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, superAdminToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: rootAdminID, Subject: superAdminEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	or, err := svc.CreateOrg(context.Background(), ownerToken, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	err = svc.CreateOrgMemberships(context.Background(), ownerToken, or.ID, memberships...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	err = svc.AssignRole(context.Background(), rootAdminID, auth.RoleRootAdmin)
	require.Nil(t, err, fmt.Sprintf("saving role expected to succeed: %s", err))

	orgRes := auth.Org{
		ID:          or.ID,
		OwnerID:     or.OwnerID,
		Name:        or.Name,
		Description: or.Description,
	}

	cases := []struct {
		desc  string
		token string
		orgID string
		org   auth.Org
		err   error
	}{
		{
			desc:  "view org as owner",
			token: ownerToken,
			orgID: or.ID,
			org:   orgRes,
			err:   nil,
		},
		{
			desc:  "view org as viewer",
			token: viewerToken,
			orgID: or.ID,
			org:   orgRes,
			err:   nil,
		},
		{
			desc:  "view org as editor",
			token: editorToken,
			orgID: or.ID,
			org:   orgRes,
			err:   nil,
		},
		{
			desc:  "view org as admin",
			token: adminToken,
			orgID: or.ID,
			org:   orgRes,
			err:   nil,
		},
		{
			desc:  "view org as system admin",
			token: superAdminToken,
			orgID: or.ID,
			org:   orgRes,
			err:   nil,
		},
		{
			desc:  "view org with wrong credentials",
			token: invalid,
			orgID: or.ID,
			org:   auth.Org{},
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "view org without credentials",
			token: "",
			orgID: or.ID,
			org:   auth.Org{},
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "view org without ID",
			token: ownerToken,
			orgID: "",
			org:   auth.Org{},
			err:   dbutil.ErrNotFound,
		},
		{
			desc:  "view non-existing org",
			token: ownerToken,
			orgID: invalid,
			org:   auth.Org{},
			err:   dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		res, err := svc.ViewOrg(context.Background(), tc.token, tc.orgID)
		org := auth.Org{
			ID:          res.ID,
			OwnerID:     res.OwnerID,
			Name:        res.Name,
			Description: res.Description,
		}
		assert.Equal(t, tc.org, org, fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.org, org))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestCreateOrgMemberships(t *testing.T) {
	svc := newService()

	_, ownerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: ownerID, Subject: ownerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, viewerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: viewerID, Subject: viewerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, editorToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: editorID, Subject: editorEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, adminToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: adminID, Subject: adminEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	or, err := svc.CreateOrg(context.Background(), ownerToken, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	mb := []auth.OrgMembership{
		{
			MemberID: "example1",
			Email:    "example1@test.com",
			Role:     auth.Viewer,
		},
		{
			MemberID: "example2",
			Email:    "example2@test.com",
			Role:     auth.Viewer,
		},
	}
	cases := []struct {
		desc        string
		token       string
		orgID       string
		memberships []auth.OrgMembership
		err         error
	}{
		{
			desc:        "create org memberships as owner",
			token:       ownerToken,
			orgID:       or.ID,
			memberships: memberships,
			err:         nil,
		},
		{
			desc:        "create org memberships as admin",
			token:       adminToken,
			orgID:       or.ID,
			memberships: mb,
			err:         nil,
		},
		{
			desc:        "create org memberships as editor",
			token:       editorToken,
			orgID:       or.ID,
			memberships: mb,
			err:         errors.ErrAuthorization,
		},
		{
			desc:        "create org memberships as viewer",
			token:       viewerToken,
			orgID:       or.ID,
			memberships: mb,
			err:         errors.ErrAuthorization,
		},
		{
			desc:        "create org memberships with wrong credentials",
			token:       invalid,
			orgID:       or.ID,
			memberships: mb,
			err:         errors.ErrAuthentication,
		},
		{
			desc:        "create org memberships without credentials",
			token:       "",
			orgID:       or.ID,
			memberships: mb,
			err:         errors.ErrAuthentication,
		},
		{
			desc:        "create org memberships without org id",
			token:       ownerToken,
			orgID:       "",
			memberships: memberships,
			err:         dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.CreateOrgMemberships(context.Background(), tc.token, tc.orgID, tc.memberships...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRemoveOrgMemberships(t *testing.T) {
	svc := newService()

	_, ownerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: ownerID, Subject: ownerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, viewerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: viewerID, Subject: viewerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, editorToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: editorID, Subject: editorEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, adminToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: adminID, Subject: adminEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	or, err := svc.CreateOrg(context.Background(), ownerToken, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	err = svc.CreateOrgMemberships(context.Background(), ownerToken, or.ID, memberships...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := []struct {
		desc     string
		token    string
		orgID    string
		memberID string
		err      error
	}{
		{
			desc:     "remove org membership as viewer",
			token:    viewerToken,
			orgID:    or.ID,
			memberID: editorID,
			err:      errors.ErrAuthorization,
		},
		{
			desc:     "remove org membership as editor",
			token:    editorToken,
			orgID:    or.ID,
			memberID: viewerID,
			err:      errors.ErrAuthorization,
		},
		{
			desc:     "remove org membership as admin",
			token:    adminToken,
			orgID:    or.ID,
			memberID: viewerID,
			err:      nil,
		},
		{
			desc:     "remove owner from org as admin",
			token:    adminToken,
			orgID:    or.ID,
			memberID: ownerID,
			err:      errors.ErrAuthorization,
		},
		{
			desc:     "remove org membership as owner",
			token:    ownerToken,
			orgID:    or.ID,
			memberID: editorID,
			err:      nil,
		},
		{
			desc:     "remove org membership with wrong credentials",
			token:    invalid,
			orgID:    or.ID,
			memberID: editorID,
			err:      errors.ErrAuthentication,
		},
		{
			desc:     "remove org membership without credentials",
			token:    "",
			orgID:    or.ID,
			memberID: editorID,
			err:      errors.ErrAuthentication,
		},
		{
			desc:     "remove membership from non-existing org",
			token:    ownerToken,
			orgID:    invalid,
			memberID: editorID,
			err:      dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveOrgMemberships(context.Background(), tc.token, tc.orgID, tc.memberID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateOrgMemberships(t *testing.T) {
	svc := newService()

	_, ownerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: ownerID, Subject: ownerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, viewerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: viewerID, Subject: viewerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, editorToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: editorID, Subject: editorEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, adminToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: adminID, Subject: adminEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	or, err := svc.CreateOrg(context.Background(), ownerToken, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	err = svc.CreateOrgMemberships(context.Background(), ownerToken, or.ID, memberships...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	orgOwner := auth.OrgMembership{Email: ownerEmail, Role: auth.Owner}

	cases := []struct {
		desc       string
		token      string
		orgID      string
		membership auth.OrgMembership
		err        error
	}{
		{
			desc:       "update org membership as viewer",
			token:      viewerToken,
			orgID:      or.ID,
			membership: memberships[1],
			err:        errors.ErrAuthorization,
		},
		{
			desc:       "update org membership as editor",
			token:      editorToken,
			orgID:      or.ID,
			membership: memberships[2],
			err:        errors.ErrAuthorization,
		},
		{
			desc:       "update org membership as admin",
			token:      adminToken,
			orgID:      or.ID,
			membership: memberships[2],
			err:        nil,
		},
		{
			desc:       "update org membership as owner",
			token:      ownerToken,
			orgID:      or.ID,
			membership: memberships[1],
			err:        nil,
		},
		{
			desc:       "update org owner role as owner",
			token:      ownerToken,
			orgID:      or.ID,
			membership: orgOwner,
			err:        errors.ErrAuthorization,
		},
		{
			desc:       "update org owner role as admin",
			token:      adminToken,
			orgID:      or.ID,
			membership: orgOwner,
			err:        errors.ErrAuthorization,
		},
		{
			desc:       "update org membership with wrong credentials",
			token:      invalid,
			orgID:      or.ID,
			membership: memberships[1],
			err:        errors.ErrAuthentication,
		},
		{
			desc:       "update org membership without credentials",
			token:      "",
			orgID:      or.ID,
			membership: memberships[1],
			err:        errors.ErrAuthentication,
		},
		{
			desc:       "update org membership with non-existing org",
			token:      ownerToken,
			orgID:      invalid,
			membership: memberships[1],
			err:        dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.UpdateOrgMemberships(context.Background(), tc.token, tc.orgID, tc.membership)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestListOrgMemberships(t *testing.T) {
	svc := newService()

	_, ownerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: ownerID, Subject: ownerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, viewerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: viewerID, Subject: viewerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, editorToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: editorID, Subject: editorEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, adminToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: adminID, Subject: adminEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, superAdminToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: rootAdminID, Subject: superAdminEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	or, err := svc.CreateOrg(context.Background(), ownerToken, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	err = svc.CreateOrgMemberships(context.Background(), ownerToken, or.ID, memberships...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	var n uint64 = 4

	err = svc.AssignRole(context.Background(), rootAdminID, auth.RoleRootAdmin)
	require.Nil(t, err, fmt.Sprintf("saving role expected to succeed: %s", err))

	cases := []struct {
		desc  string
		token string
		orgID string
		meta  apiutil.PageMetadata
		size  uint64
		err   error
	}{
		{
			desc:  "list org memberships as owner",
			token: ownerToken,
			orgID: or.ID,
			meta: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		},
		{
			desc:  "list org memberships as admin",
			token: adminToken,
			orgID: or.ID,
			meta: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		},
		{
			desc:  "list org memberships as editor",
			token: editorToken,
			orgID: or.ID,
			meta: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		},
		{
			desc:  "list org memberships as viewer",
			token: viewerToken,
			orgID: or.ID,
			meta: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		},
		{
			desc:  "list org memberships as system admin",
			token: superAdminToken,
			orgID: or.ID,
			meta: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		},
		{
			desc:  "list half org memberships",
			token: viewerToken,
			orgID: or.ID,
			meta: apiutil.PageMetadata{
				Offset: 0,
				Limit:  n / 2,
			},
			size: n / 2,
			err:  nil,
		},
		{
			desc:  "list last org membership",
			token: viewerToken,
			orgID: or.ID,
			meta: apiutil.PageMetadata{
				Offset: n - 1,
				Limit:  1,
			},
			size: 1,
			err:  nil,
		},
		{
			desc:  "list org memberships with wrong credentials",
			token: invalid,
			orgID: or.ID,
			meta:  apiutil.PageMetadata{},
			size:  0,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "list org memberships without credentials",
			token: "",
			orgID: or.ID,
			meta:  apiutil.PageMetadata{},
			size:  0,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "list memberships from non-existing org",
			token: ownerToken,
			orgID: invalid,
			meta:  apiutil.PageMetadata{},
			size:  0,
			err:   dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		page, err := svc.ListOrgMemberships(context.Background(), tc.token, tc.orgID, tc.meta)
		size := uint64(len(page.OrgMemberships))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s expected %d got %d\n", tc.desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestCreateOrgInvite(t *testing.T) {
	svc := newService()

	_, ownerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: ownerID, Subject: ownerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, viewerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: viewerID, Subject: viewerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, editorToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: editorID, Subject: editorEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, adminToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: adminID, Subject: adminEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, superAdminToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: rootAdminID, Subject: superAdminEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	or, err := svc.CreateOrg(context.Background(), ownerToken, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	err = svc.CreateOrgMemberships(context.Background(), ownerToken, or.ID, memberships...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	err = svc.AssignRole(context.Background(), rootAdminID, auth.RoleRootAdmin)
	require.Nil(t, err, fmt.Sprintf("saving role expected to succeed: %s", err))

	cases := []struct {
		desc       string
		token      string
		orgID      string
		membership auth.OrgMembership
		err        error
	}{
		{
			desc:  "create org invite as root admin",
			token: superAdminToken,
			orgID: or.ID,
			membership: auth.OrgMembership{
				Role:  auth.Viewer,
				Email: "example1@test.com",
			},
			err: nil,
		},
		{
			desc:  "create org invite as org owner",
			token: ownerToken,
			orgID: or.ID,
			membership: auth.OrgMembership{
				Role:  auth.Viewer,
				Email: "example2@test.com",
			},
			err: nil,
		},
		{
			desc:  "create org invite as org admin",
			token: adminToken,
			orgID: or.ID,
			membership: auth.OrgMembership{
				Role:  auth.Viewer,
				Email: "example3@test.com",
			},
			err: nil,
		},
		{
			desc:  "create org invite as org editor",
			token: editorToken,
			orgID: or.ID,
			membership: auth.OrgMembership{
				Role:  auth.Viewer,
				Email: "example4@test.com",
			},
			err: errors.ErrAuthorization,
		},
		{
			desc:  "create org invite as org viewer",
			token: viewerToken,
			orgID: or.ID,
			membership: auth.OrgMembership{
				Role:  auth.Viewer,
				Email: "example5@test.com",
			},
			err: errors.ErrAuthorization,
		},
		{
			desc:  "create org invite with pending invite to same org",
			token: adminToken,
			orgID: or.ID,
			membership: auth.OrgMembership{
				Role:  auth.Viewer,
				Email: "example3@test.com",
			},
			err: dbutil.ErrConflict,
		},
		{
			desc:  "create org invite towards invitee who is already a member of org",
			token: adminToken,
			orgID: or.ID,
			membership: auth.OrgMembership{
				Role:  auth.Viewer,
				Email: editorEmail,
			},
			err: auth.ErrOrgMembershipExists,
		},
		{
			desc:  "create org invite towards invitee with unregistered e-mail",
			token: adminToken,
			orgID: or.ID,
			membership: auth.OrgMembership{
				Role:  auth.Editor,
				Email: unregisteredEmail,
			},
			err: dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		_, err := svc.CreateOrgInvite(context.Background(), tc.token, auth.OrgInviteRequest{
			Email:        tc.membership.Email,
			Role:         tc.membership.Role,
			OrgID:        tc.orgID,
			RedirectPath: redirectPathInvite,
		})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRevokeInvite(t *testing.T) {
	svc := newService()

	_, ownerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: ownerID, Subject: ownerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	invitee := usersByEmails["example1@test.com"]

	_, thirdToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: usersByEmails["example2@test.com"].ID, Subject: ownerEmail})
	assert.Nil(t, err, fmt.Sprintf("unexpected error issuing login token: %s\n", err))

	testOrg, err := svc.CreateOrg(context.Background(), ownerToken, org)
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	testInvite, err := svc.CreateOrgInvite(context.Background(), ownerToken, auth.OrgInviteRequest{
		Email:        invitee.Email,
		Role:         auth.Viewer,
		OrgID:        testOrg.ID,
		RedirectPath: redirectPathInvite,
	})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	testInviteID := testInvite.ID

	cases := []struct {
		desc  string
		token string
		err   error
	}{
		{
			desc:  "revoke invite as unauthorized user",
			token: thirdToken,
			err:   errors.ErrAuthorization,
		},
		{
			desc:  "revoke invite as inviter",
			token: ownerToken,
			err:   nil,
		},
	}

	for _, tc := range cases {
		err := svc.RevokeOrgInvite(context.Background(), tc.token, testInviteID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}

}

func TestRespondInvite(t *testing.T) {
	svc := newService()

	_, ownerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: ownerID, Subject: ownerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	_, unauthToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: "example4", Subject: "example4@test.com"})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	_, example1Token, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: "example1", Subject: "example1@test.com"})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	_, example2Token, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: "example2", Subject: "example2@test.com"})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	testOrg, err := svc.CreateOrg(context.Background(), ownerToken, org)
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	testInvites := []auth.OrgInvite{}
	for i := range 3 {
		inv, err := svc.CreateOrgInvite(context.Background(), ownerToken, auth.OrgInviteRequest{
			Email:        fmt.Sprintf("example%d@test.com", i+1),
			Role:         auth.Viewer,
			OrgID:        testOrg.ID,
			RedirectPath: redirectPathInvite,
		})

		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
		testInvites = append(testInvites, inv)
	}

	cases := []struct {
		desc     string
		token    string
		inviteID string
		accept   bool
		err      error
	}{
		{
			desc:     "respond to invite as unauthorized user",
			token:    unauthToken,
			inviteID: testInvites[0].ID,
			accept:   true,
			err:      errors.ErrAuthorization,
		},
		{
			desc:     "accept invite",
			token:    example1Token,
			inviteID: testInvites[0].ID,
			accept:   true,
			err:      nil,
		},
		{
			desc:     "decline invite",
			token:    example2Token,
			inviteID: testInvites[1].ID,
			accept:   false,
			err:      nil,
		},
	}

	for _, tc := range cases {
		err := svc.RespondOrgInvite(context.Background(), tc.token, tc.inviteID, tc.accept)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected: %s, got: %s\n", tc.desc, tc.err, err))
	}
}

func TestViewInvite(t *testing.T) {
	svc := newService()

	_, ownerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: ownerID, Subject: ownerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	err = svc.AssignRole(context.Background(), rootAdminID, auth.RoleRootAdmin)
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	_, rootAdminToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: rootAdminID, Subject: superAdminEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	testOrg, err := svc.CreateOrg(context.Background(), ownerToken, org)
	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	inviter := usersByEmails["example1@test.com"]
	invitee := usersByEmails["example2@test.com"]

	_, inviterToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: inviter.ID, Subject: inviter.Email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	_, inviteeToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: invitee.ID, Subject: invitee.Email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	_, unauthToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: "example3", Subject: "example3@email.com"})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	err = svc.CreateOrgMemberships(context.Background(), ownerToken, testOrg.ID, auth.OrgMembership{
		MemberID: inviter.ID,
		Email:    inviter.Email,
		Role:     auth.RoleAdmin,
	})

	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	invite, err := svc.CreateOrgInvite(context.Background(), inviterToken, auth.OrgInviteRequest{
		Email:        invitee.Email,
		Role:         auth.Viewer,
		OrgID:        testOrg.ID,
		RedirectPath: redirectPathInvite,
	})

	assert.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := []struct {
		desc  string
		token string
		err   error
	}{
		{
			desc:  "view invite as invitee",
			token: inviteeToken,
			err:   nil,
		},
		{
			desc:  "view invite as inviter",
			token: inviterToken,
			err:   nil,
		},
		{
			desc:  "view invite as root admin",
			token: rootAdminToken,
			err:   nil,
		},
		{
			desc:  "view invite as unauthorized user",
			token: unauthToken,
			err:   errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		_, err := svc.ViewOrgInvite(context.Background(), tc.token, invite.ID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected :%s, got :%s\n", tc.desc, tc.err, err))
	}

}

func TestListInvitesByUser(t *testing.T) {
	svc := newService()

	_, ownerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: ownerID, Subject: ownerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	err = svc.AssignRole(context.Background(), rootAdminID, auth.RoleRootAdmin)
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	_, rootAdminToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: rootAdminID, Subject: superAdminEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	invitee := usersByEmails["example1@test.com"]

	_, inviteeToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: invitee.ID, Subject: invitee.Email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	_, unauthToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: "example3", Subject: "example3@test.com"})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	n := uint64(5)

	for i := uint64(1); i <= n; i++ {
		org, err := svc.CreateOrg(context.Background(), ownerToken, auth.Org{
			Name: fmt.Sprintf("org%d", i),
		})

		assert.Nil(t, err, fmt.Sprintf("Creating Org expected to succeed: %s", err))

		_, err = svc.CreateOrgInvite(context.Background(), ownerToken, auth.OrgInviteRequest{
			Email:        invitee.Email,
			Role:         auth.Viewer,
			OrgID:        org.ID,
			RedirectPath: redirectPathInvite,
		})

		assert.Nil(t, err, fmt.Sprintf("Unexpected error inviting Org member: %s", err))
	}

	cases := []struct {
		desc     string
		token    string
		pm       auth.PageMetadataInvites
		userID   string
		userType string
		size     uint64
		err      error
	}{
		{
			desc:     "list all pending invites as invitee",
			token:    inviteeToken,
			pm:       auth.PageMetadataInvites{PageMetadata: apiutil.PageMetadata{Limit: n, Offset: 0}},
			userID:   invitee.ID,
			userType: auth.UserTypeInvitee,
			size:     n,
			err:      nil,
		},
		{
			desc:     "list all pending invites as root admin",
			token:    rootAdminToken,
			pm:       auth.PageMetadataInvites{PageMetadata: apiutil.PageMetadata{Limit: n, Offset: 0}},
			userID:   invitee.ID,
			userType: auth.UserTypeInvitee,
			size:     n,
			err:      nil,
		},
		{
			desc:     "list all pending invites as unauthorized user",
			token:    unauthToken,
			pm:       auth.PageMetadataInvites{PageMetadata: apiutil.PageMetadata{Limit: n, Offset: 0}},
			userID:   invitee.ID,
			userType: auth.UserTypeInvitee,
			size:     0,
			err:      errors.ErrAuthorization,
		},
		{
			desc:     "list half of pending invites as invitee",
			token:    inviteeToken,
			pm:       auth.PageMetadataInvites{PageMetadata: apiutil.PageMetadata{Limit: n / 2, Offset: 0}},
			userID:   invitee.ID,
			userType: auth.UserTypeInvitee,
			size:     n / 2,
			err:      nil,
		},
		{
			desc:     "list last pending invite as invitee",
			token:    inviteeToken,
			pm:       auth.PageMetadataInvites{PageMetadata: apiutil.PageMetadata{Limit: 1, Offset: n - 1}},
			userID:   invitee.ID,
			userType: auth.UserTypeInvitee,
			size:     1,
			err:      nil,
		},
		{
			desc:     "list all sent invites as inviter",
			token:    ownerToken,
			pm:       auth.PageMetadataInvites{PageMetadata: apiutil.PageMetadata{Limit: n, Offset: 0}},
			userID:   ownerID,
			userType: auth.UserTypeInviter,
			size:     n,
			err:      nil,
		},
		{
			desc:     "list all sent invites as root admin",
			token:    rootAdminToken,
			pm:       auth.PageMetadataInvites{PageMetadata: apiutil.PageMetadata{Limit: n, Offset: 0}},
			userID:   ownerID,
			userType: auth.UserTypeInviter,
			size:     n,
			err:      nil,
		},
		{
			desc:     "list all sent invites as unauthorized user",
			token:    unauthToken,
			pm:       auth.PageMetadataInvites{PageMetadata: apiutil.PageMetadata{Limit: n, Offset: 0}},
			userID:   ownerID,
			userType: auth.UserTypeInviter,
			size:     0,
			err:      errors.ErrAuthorization,
		},
		{
			desc:     "list half of sent invites as inviter",
			token:    ownerToken,
			pm:       auth.PageMetadataInvites{PageMetadata: apiutil.PageMetadata{Limit: n / 2, Offset: 0}},
			userID:   ownerID,
			userType: auth.UserTypeInviter,
			size:     n / 2,
			err:      nil,
		},
		{
			desc:     "list last sent invite as inviter",
			token:    ownerToken,
			pm:       auth.PageMetadataInvites{PageMetadata: apiutil.PageMetadata{Limit: 1, Offset: n - 1}},
			userID:   ownerID,
			userType: auth.UserTypeInviter,
			size:     1,
			err:      nil,
		},
	}

	for _, tc := range cases {
		invitesPage, err := svc.ListOrgInvitesByUser(context.Background(), tc.token, tc.userType, tc.userID, tc.pm)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))

		invCount := uint64(len(invitesPage.Invites))

		assert.Equal(t, invCount, tc.size, fmt.Sprintf("%s: expected %d elements, got %d\n", tc.desc, tc.size, invCount))
	}
}

func TestBackup(t *testing.T) {
	svc := newService()

	_, ownerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: ownerID, Subject: ownerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, viewerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: viewerID, Subject: viewerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, superAdminToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: rootAdminID, Subject: superAdminEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	var grIDs []string
	for i := 0; i < n; i++ {
		groupID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
		grIDs = append(grIDs, groupID)
	}

	or, err := svc.CreateOrg(context.Background(), ownerToken, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	err = svc.CreateOrgMemberships(context.Background(), ownerToken, or.ID, memberships...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	err = svc.AssignRole(context.Background(), rootAdminID, auth.RoleRootAdmin)
	require.Nil(t, err, fmt.Sprintf("saving role expected to succeed: %s", err))

	cases := []struct {
		desc              string
		token             string
		orgSize           int
		orgMembershipSize int
		err               error
	}{
		{
			desc:              "backup all orgs, org memberships and org groups",
			token:             superAdminToken,
			orgSize:           1,
			orgMembershipSize: len(memberships) + 1,
			err:               nil,
		},
		{
			desc:              "backup with invalid credentials",
			token:             invalid,
			orgSize:           0,
			orgMembershipSize: 0,
			err:               errors.ErrAuthentication,
		},
		{
			desc:              "backup without credentials",
			token:             "",
			orgSize:           0,
			orgMembershipSize: 0,
			err:               errors.ErrAuthentication,
		},
		{
			desc:              "backup with unauthorised credentials",
			token:             viewerToken,
			orgSize:           0,
			orgMembershipSize: 0,
			err:               errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		page, err := svc.Backup(context.Background(), tc.token)
		orgSize := len(page.Orgs)
		orgMembershipSize := len(page.OrgMemberships)
		assert.Equal(t, tc.orgSize, orgSize, fmt.Sprintf("%s expected %d got %d\n", tc.desc, tc.orgSize, orgSize))
		assert.Equal(t, tc.orgMembershipSize, orgMembershipSize, fmt.Sprintf("%s expected %d got %d\n", tc.desc, tc.orgMembershipSize, orgMembershipSize))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRestore(t *testing.T) {
	svc := newService()

	_, superAdminToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: rootAdminID, Subject: superAdminEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, viewerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: viewerID, Subject: viewerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	err = svc.AssignRole(context.Background(), rootAdminID, auth.RoleRootAdmin)
	require.Nil(t, err, fmt.Sprintf("saving role expected to succeed: %s", err))

	var memberIDs []string
	var groupIDs []string
	for i := 0; i < n; i++ {
		memberID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
		memberIDs = append(memberIDs, memberID)

		groupID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
		groupIDs = append(groupIDs, groupID)
	}

	orgs := []auth.Org{{ID: id, OwnerID: ownerID, Name: name}}
	var orgMemberships []auth.OrgMembership
	for _, memberID := range memberIDs {
		orgMemberships = append(orgMemberships, auth.OrgMembership{MemberID: memberID, OrgID: id})
	}

	backup := auth.Backup{
		Orgs:           orgs,
		OrgMemberships: orgMemberships,
	}

	cases := []struct {
		desc   string
		token  string
		backup auth.Backup
		err    error
	}{
		{
			desc:   "restore all orgs, org memberships and org groups",
			token:  superAdminToken,
			backup: backup,
			err:    nil,
		},
		{
			desc:   "restore with invalid credentials",
			token:  invalid,
			backup: backup,
			err:    errors.ErrAuthentication,
		},
		{
			desc:   "restore without credentials",
			token:  "",
			backup: backup,
			err:    errors.ErrAuthentication,
		},
		{
			desc:   "restore with unauthorised credentials",
			token:  viewerToken,
			backup: backup,
			err:    errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		err := svc.Restore(context.Background(), tc.token, backup)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}
