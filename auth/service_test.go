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
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	thmocks "github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	secret          = "secret"
	email           = "test@example.com"
	superAdminEmail = "admin@example.com"
	ownerEmail      = "owner@test.com"
	viewerEmail     = "viewer@test.com"
	editorEmail     = "editor@test.com"
	adminEmail      = "admin@test.com"
	id              = "testID"
	ownerID         = "ownerID"
	adminID         = "adminID"
	editorID        = "editorID"
	viewerID        = "viewerID"
	rootAdminID     = "rootAdminID"
	description     = "description"
	name            = "name"
	invalid         = "invalid"
	n               = 10

	loginDuration = 30 * time.Minute
)

var (
	org           = auth.Org{Name: name, Description: description}
	members       = []auth.OrgMember{{Email: adminEmail, Role: auth.Admin}, {Email: editorEmail, Role: auth.Editor}, {Email: viewerEmail, Role: auth.Viewer}}
	usersByEmails = map[string]users.User{adminEmail: {ID: adminID, Email: adminEmail}, editorEmail: {ID: editorID, Email: editorEmail}, viewerEmail: {ID: viewerID, Email: viewerEmail}, ownerEmail: {ID: ownerID, Email: ownerEmail}}
	usersByIDs    = map[string]users.User{adminID: {ID: adminID, Email: adminEmail}, editorID: {ID: editorID, Email: editorEmail}, viewerID: {ID: viewerID, Email: viewerEmail}, ownerID: {ID: ownerID, Email: ownerEmail}}
	idProvider    = uuid.New()
)

func newService() auth.Service {
	keyRepo := mocks.NewKeyRepository()
	idMockProvider := uuid.NewMock()
	orgRepo := mocks.NewOrgRepository()
	roleRepo := mocks.NewRolesRepository()
	uc := mocks.NewUsersService(usersByIDs, usersByEmails)
	tc := thmocks.NewThingsServiceClient(nil, nil, createGroups())
	t := jwt.New(secret)
	return auth.New(orgRepo, tc, uc, keyRepo, roleRepo, idMockProvider, t, loginDuration)
}

func createGroups() map[string]things.Group {
	groups := make(map[string]things.Group, n)
	for i := 0; i < n; i++ {
		groupId := fmt.Sprintf(id+"-%d", i)
		groups[groupId] = things.Group{
			ID:          groupId,
			OwnerID:     ownerID,
			Name:        fmt.Sprintf(name+"-%d", i),
			Description: fmt.Sprintf(description+"-%d", i),
			Metadata:    map[string]interface{}{"meta": "data"},
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
			err:   errors.ErrNotFound,
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

	pr := auth.AuthzReq{Token: adminToken, Subject: auth.RootSubject}
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
		name := fmt.Sprintf("org-%d", i)
		description := fmt.Sprintf("description-%d", i)
		org.Name = name
		org.Description = description
		_, err := svc.CreateOrg(context.Background(), ownerToken, org)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	}

	err = svc.AssignRole(context.Background(), rootAdminID, auth.RoleRootAdmin)
	require.Nil(t, err, fmt.Sprintf("saving role expected to succeed: %s", err))

	cases := []struct {
		desc  string
		token string
		meta  auth.PageMetadata
		size  uint64
		err   error
	}{
		{
			desc:  "list orgs",
			token: ownerToken,
			meta: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		}, {
			desc:  "list orgs as system admin",
			token: superAdminToken,
			meta: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		}, {
			desc:  "list orgs with wrong credentials",
			token: invalid,
			meta:  auth.PageMetadata{},
			size:  0,
			err:   errors.ErrAuthentication,
		}, {
			desc:  "list orgs without credentials",
			token: "",
			meta:  auth.PageMetadata{},
			size:  0,
			err:   errors.ErrAuthentication,
		}, {
			desc:  "list half of total orgs",
			token: ownerToken,
			meta: auth.PageMetadata{
				Offset: n / 2,
				Limit:  n,
			},
			size: n / 2,
			err:  nil,
		}, {
			desc:  "list last org",
			token: ownerToken,
			meta: auth.PageMetadata{
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

	err = svc.AssignMembers(context.Background(), ownerToken, res.ID, members...)
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
			err:   errors.ErrNotFound,
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
			err:   errors.ErrNotFound,
		},
		{
			desc:  "remove non-existing org",
			token: ownerToken,
			id:    invalid,
			err:   errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveOrg(context.Background(), tc.token, tc.id)
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

	err = svc.AssignMembers(context.Background(), ownerToken, res.ID, members...)
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
			err:   errors.ErrNotFound,
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

	err = svc.AssignMembers(context.Background(), ownerToken, or.ID, members...)
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
			err:   errors.ErrNotFound,
		},
		{
			desc:  "view non-existing org",
			token: ownerToken,
			orgID: invalid,
			org:   auth.Org{},
			err:   errors.ErrNotFound,
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

func TestAssignMembers(t *testing.T) {
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

	mb := []auth.OrgMember{
		{
			MemberID: "member1",
			Role:     auth.Viewer,
		},
		{
			MemberID: "member2",
			Role:     auth.Viewer,
		},
	}
	cases := []struct {
		desc   string
		token  string
		orgID  string
		member []auth.OrgMember
		err    error
	}{
		{
			desc:   "assign members to org as owner",
			token:  ownerToken,
			orgID:  or.ID,
			member: members,
			err:    nil,
		},
		{
			desc:   "assign members to org as admin",
			token:  adminToken,
			orgID:  or.ID,
			member: mb,
			err:    nil,
		},
		{
			desc:   "assign members to org as editor",
			token:  editorToken,
			orgID:  or.ID,
			member: mb,
			err:    errors.ErrAuthorization,
		},
		{
			desc:   "assign members to org as viewer",
			token:  viewerToken,
			orgID:  or.ID,
			member: mb,
			err:    errors.ErrAuthorization,
		},
		{
			desc:   "assign members with wrong credentials",
			token:  invalid,
			orgID:  or.ID,
			member: mb,
			err:    errors.ErrAuthentication,
		},
		{
			desc:   "assign members without credentials",
			token:  "",
			orgID:  or.ID,
			member: mb,
			err:    errors.ErrAuthentication,
		},
		{
			desc:   "assign members to non-existing org",
			token:  ownerToken,
			orgID:  invalid,
			member: members,
			err:    errors.ErrNotFound,
		},
		{
			desc:   "assign members to org without id",
			token:  ownerToken,
			orgID:  "",
			member: members,
			err:    errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.AssignMembers(context.Background(), tc.token, tc.orgID, tc.member...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUnassignMembers(t *testing.T) {
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

	err = svc.AssignMembers(context.Background(), ownerToken, or.ID, members...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := []struct {
		desc     string
		token    string
		orgID    string
		memberID string
		err      error
	}{
		{
			desc:     "unassign member from org as viewer",
			token:    viewerToken,
			orgID:    or.ID,
			memberID: editorID,
			err:      errors.ErrAuthorization,
		},
		{
			desc:     "unassign member from org as editor",
			token:    editorToken,
			orgID:    or.ID,
			memberID: viewerID,
			err:      errors.ErrAuthorization,
		},
		{
			desc:     "unassign member from org as admin",
			token:    adminToken,
			orgID:    or.ID,
			memberID: viewerID,
			err:      nil,
		},
		{
			desc:     "unassign owner from org as admin",
			token:    adminToken,
			orgID:    or.ID,
			memberID: ownerID,
			err:      errors.ErrAuthorization,
		},
		{
			desc:     "unassign member from org as owner",
			token:    ownerToken,
			orgID:    or.ID,
			memberID: editorID,
			err:      nil,
		},
		{
			desc:     "unassign member with wrong credentials",
			token:    invalid,
			orgID:    or.ID,
			memberID: editorID,
			err:      errors.ErrAuthentication,
		},
		{
			desc:     "unassign member without credentials",
			token:    "",
			orgID:    or.ID,
			memberID: editorID,
			err:      errors.ErrAuthentication,
		},
		{
			desc:     "unassign member from non-existing org",
			token:    ownerToken,
			orgID:    invalid,
			memberID: editorID,
			err:      errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.UnassignMembers(context.Background(), tc.token, tc.orgID, tc.memberID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUpdateMembers(t *testing.T) {
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

	err = svc.AssignMembers(context.Background(), ownerToken, or.ID, members...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	orgOwner := auth.OrgMember{Email: ownerEmail, Role: auth.Owner}

	cases := []struct {
		desc   string
		token  string
		orgID  string
		member auth.OrgMember
		err    error
	}{
		{
			desc:   "update org member role as viewer",
			token:  viewerToken,
			orgID:  or.ID,
			member: members[1],
			err:    errors.ErrAuthorization,
		},
		{
			desc:   "update org member role as editor",
			token:  editorToken,
			orgID:  or.ID,
			member: members[2],
			err:    errors.ErrAuthorization,
		},
		{
			desc:   "update org member role as admin",
			token:  adminToken,
			orgID:  or.ID,
			member: members[2],
			err:    nil,
		},
		{
			desc:   "update org member role as owner",
			token:  ownerToken,
			orgID:  or.ID,
			member: members[1],
			err:    nil,
		},
		{
			desc:   "update org owner role as owner",
			token:  ownerToken,
			orgID:  or.ID,
			member: orgOwner,
			err:    errors.ErrAuthorization,
		},
		{
			desc:   "update org owner role as admin",
			token:  adminToken,
			orgID:  or.ID,
			member: orgOwner,
			err:    errors.ErrAuthorization,
		},
		{
			desc:   "update org member role with wrong credentials",
			token:  invalid,
			orgID:  or.ID,
			member: members[1],
			err:    errors.ErrAuthentication,
		},
		{
			desc:   "update org member role without credentials",
			token:  "",
			orgID:  or.ID,
			member: members[1],
			err:    errors.ErrAuthentication,
		},
		{
			desc:   "update org member role with non-existing org",
			token:  ownerToken,
			orgID:  invalid,
			member: members[1],
			err:    errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.UpdateMembers(context.Background(), tc.token, tc.orgID, tc.member)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestListMembersByOrg(t *testing.T) {
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

	err = svc.AssignMembers(context.Background(), ownerToken, or.ID, members...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	var n uint64 = 4

	err = svc.AssignRole(context.Background(), rootAdminID, auth.RoleRootAdmin)
	require.Nil(t, err, fmt.Sprintf("saving role expected to succeed: %s", err))

	cases := []struct {
		desc  string
		token string
		orgID string
		meta  auth.PageMetadata
		size  uint64
		err   error
	}{
		{
			desc:  "list org members as owner",
			token: ownerToken,
			orgID: or.ID,
			meta: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		},
		{
			desc:  "list org members as admin",
			token: adminToken,
			orgID: or.ID,
			meta: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		},
		{
			desc:  "list org members as editor",
			token: editorToken,
			orgID: or.ID,
			meta: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		},
		{
			desc:  "list org members as viewer",
			token: viewerToken,
			orgID: or.ID,
			meta: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		},
		{
			desc:  "list org members as system admin",
			token: superAdminToken,
			orgID: or.ID,
			meta: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		},
		{
			desc:  "list half org members",
			token: viewerToken,
			orgID: or.ID,
			meta: auth.PageMetadata{
				Offset: 0,
				Limit:  n / 2,
			},
			size: n / 2,
			err:  nil,
		},
		{
			desc:  "list last org member",
			token: viewerToken,
			orgID: or.ID,
			meta: auth.PageMetadata{
				Offset: n - 1,
				Limit:  1,
			},
			size: 1,
			err:  nil,
		},
		{
			desc:  "list org members with wrong credentials",
			token: invalid,
			orgID: or.ID,
			meta:  auth.PageMetadata{},
			size:  0,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "list org members without credentials",
			token: "",
			orgID: or.ID,
			meta:  auth.PageMetadata{},
			size:  0,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "list members from non-existing org",
			token: ownerToken,
			orgID: invalid,
			meta:  auth.PageMetadata{},
			size:  0,
			err:   nil,
		},
	}

	for _, tc := range cases {
		page, err := svc.ListMembersByOrg(context.Background(), tc.token, tc.orgID, tc.meta)
		size := uint64(len(page.OrgMembers))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s expected %d got %d\n", tc.desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestListOrgsByMember(t *testing.T) {
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

	for i := 0; i < n; i++ {
		o := auth.Org{
			Name:        fmt.Sprintf("org-%d", i),
			Description: fmt.Sprintf("description-%d", i),
		}

		or, err := svc.CreateOrg(context.Background(), ownerToken, o)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
		or.Name = fmt.Sprintf("%s-%d", or.Name, i)

		err = svc.AssignMembers(context.Background(), ownerToken, or.ID, members...)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	}

	err = svc.AssignRole(context.Background(), rootAdminID, auth.RoleRootAdmin)
	require.Nil(t, err, fmt.Sprintf("saving role expected to succeed: %s", err))

	cases := []struct {
		desc     string
		token    string
		memberID string
		meta     auth.PageMetadata
		size     int
		err      error
	}{
		{
			desc:     "list member organisations as owner",
			token:    ownerToken,
			memberID: ownerID,
			meta: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		},
		{
			desc:     "list member organisations as admin",
			token:    adminToken,
			memberID: adminID,
			meta: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		},
		{
			desc:     "list member organisations as editor",
			token:    editorToken,
			memberID: editorID,
			meta: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		},
		{
			desc:     "list member organisations as viewer",
			token:    viewerToken,
			memberID: viewerID,
			meta: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		},
		{
			desc:     "list member organisations as system admin",
			token:    superAdminToken,
			memberID: ownerID,
			meta: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		},
		{
			desc:     "list half of member organisations",
			token:    viewerToken,
			memberID: viewerID,
			meta: auth.PageMetadata{
				Offset: n / 2,
				Limit:  n,
			},
			size: n / 2,
			err:  nil,
		},
		{
			desc:     "list last member organisation",
			token:    viewerToken,
			memberID: viewerID,
			meta: auth.PageMetadata{
				Offset: n - 1,
				Limit:  n,
			},
			size: 1,
			err:  nil,
		},
		{
			desc:     "list member organisations with invalid credentials",
			token:    invalid,
			memberID: viewerID,
			meta: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: 0,
			err:  errors.ErrAuthentication,
		},
		{
			desc:     "list member organisations with no credentials",
			token:    "",
			memberID: viewerID,
			meta: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: 0,
			err:  errors.ErrAuthentication,
		},
	}

	for _, tc := range cases {
		page, err := svc.ListOrgsByMember(context.Background(), tc.token, tc.memberID, tc.meta)
		size := uint64(len(page.Orgs))
		assert.Equal(t, tc.size, int(size), fmt.Sprintf("%s expected %d got %d\n", tc.desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
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

	err = svc.AssignMembers(context.Background(), ownerToken, or.ID, members...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	err = svc.AssignRole(context.Background(), rootAdminID, auth.RoleRootAdmin)
	require.Nil(t, err, fmt.Sprintf("saving role expected to succeed: %s", err))

	cases := []struct {
		desc          string
		token         string
		orgSize       int
		orgMemberSize int
		err           error
	}{
		{
			desc:          "backup all orgs, org members and org groups",
			token:         superAdminToken,
			orgSize:       1,
			orgMemberSize: len(members) + 1,
			err:           nil,
		},
		{
			desc:          "backup with invalid credentials",
			token:         invalid,
			orgSize:       0,
			orgMemberSize: 0,
			err:           errors.ErrAuthentication,
		},
		{
			desc:          "backup without credentials",
			token:         "",
			orgSize:       0,
			orgMemberSize: 0,
			err:           errors.ErrAuthentication,
		},
		{
			desc:          "backup with unauthorised credentials",
			token:         viewerToken,
			orgSize:       0,
			orgMemberSize: 0,
			err:           errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		page, err := svc.Backup(context.Background(), tc.token)
		orgSize := len(page.Orgs)
		orgMemberSize := len(page.OrgMembers)
		assert.Equal(t, tc.orgSize, orgSize, fmt.Sprintf("%s expected %d got %d\n", tc.desc, tc.orgSize, orgSize))
		assert.Equal(t, tc.orgMemberSize, orgMemberSize, fmt.Sprintf("%s expected %d got %d\n", tc.desc, tc.orgMemberSize, orgMemberSize))
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
	var orgMembers []auth.OrgMember
	for _, memberID := range memberIDs {
		orgMembers = append(orgMembers, auth.OrgMember{MemberID: memberID, OrgID: id})
	}

	var orgGroups []auth.OrgGroup
	for _, groupID := range groupIDs {
		orgGroups = append(orgGroups, auth.OrgGroup{GroupID: groupID, OrgID: id})
	}

	backup := auth.Backup{
		Orgs:       orgs,
		OrgMembers: orgMembers,
	}

	cases := []struct {
		desc   string
		token  string
		backup auth.Backup
		err    error
	}{
		{
			desc:   "restore all orgs, org members and org groups",
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
