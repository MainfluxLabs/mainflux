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
	secret      = "secret"
	email       = "test@example.com"
	viewerEmail = "viewer@example.com"
	editorEmail = "editor@example.com"
	adminEmail  = "admin@example.com"
	id          = "testID"
	ownerID     = "ownerID"
	adminID     = "adminID"
	editorID    = "editorID"
	viewerID    = "viewerID"
	description = "description"
	name        = "name"
	invalid     = "invalid"
	n           = 10

	loginDuration = 30 * time.Minute
)

var (
	org           = auth.Org{Name: name, Description: description}
	members       = []auth.Member{{Email: adminEmail, Role: auth.AdminRole}, {Email: editorEmail, Role: auth.EditorRole}, {Email: viewerEmail, Role: auth.ViewerRole}}
	usersByEmails = map[string]users.User{adminEmail: {ID: adminID, Email: adminEmail}, editorEmail: {ID: editorID, Email: editorEmail}, viewerEmail: {ID: viewerID, Email: viewerEmail}, email: {ID: id, Email: email}}
	usersByIDs    = map[string]users.User{adminID: {ID: adminID, Email: adminEmail}, editorID: {ID: editorID, Email: editorEmail}, viewerID: {ID: viewerID, Email: viewerEmail}, id: {ID: id, Email: email}}
	idProvider    = uuid.New()
)

func newService() auth.Service {
	keyRepo := mocks.NewKeyRepository()
	idMockProvider := uuid.NewMock()
	orgRepo := mocks.NewOrgRepository()
	uc := mocks.NewUsersService(usersByIDs, usersByEmails)
	tc := thmocks.NewThingsService(nil, createGroups())
	t := jwt.New(secret)
	return auth.New(orgRepo, tc, uc, keyRepo, idMockProvider, t, loginDuration, email)
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

	pr := auth.AuthzReq{Email: email}
	err := svc.Authorize(context.Background(), pr)
	require.Nil(t, err, fmt.Sprintf("authorizing initial %v authz request expected to succeed: %s", pr, err))
}

func TestCreateOrg(t *testing.T) {
	svc := newService()

	_, ownerLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	cases := []struct {
		desc  string
		token string
		org   auth.Org
		err   error
	}{
		{
			desc:  "create org",
			token: ownerLoginSecret,
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
			token: ownerLoginSecret,
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

	_, ownerLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	for i := 0; i < n; i++ {
		name := fmt.Sprintf("org-%d", i)
		description := fmt.Sprintf("description-%d", i)
		org.Name = name
		org.Description = description
		_, err := svc.CreateOrg(context.Background(), ownerLoginSecret, org)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	}

	cases := []struct {
		desc  string
		token string
		meta  auth.PageMetadata
		size  uint64
		err   error
	}{
		{
			desc:  "list orgs",
			token: ownerLoginSecret,
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
			token: ownerLoginSecret,
			meta: auth.PageMetadata{
				Offset: n / 2,
				Limit:  n,
			},
			size: n / 2,
			err:  nil,
		}, {
			desc:  "list last org",
			token: ownerLoginSecret,
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

	_, ownerLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, viewerLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: viewerID, Subject: viewerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, editorLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: editorID, Subject: editorEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, adminLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: adminID, Subject: adminEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	res, err := svc.CreateOrg(context.Background(), ownerLoginSecret, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	err = svc.AssignMembers(context.Background(), ownerLoginSecret, res.ID, members...)
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
			token: ownerLoginSecret,
			id:    invalid,
			err:   errors.ErrNotFound,
		},
		{
			desc:  "remove org as viewer",
			token: viewerLoginSecret,
			id:    res.ID,
			err:   errors.ErrAuthorization,
		},
		{
			desc:  "remove org as editor",
			token: editorLoginSecret,
			id:    res.ID,
			err:   errors.ErrAuthorization,
		},
		{
			desc:  "remove org as admin",
			token: adminLoginSecret,
			id:    res.ID,
			err:   errors.ErrAuthorization,
		},
		{
			desc:  "remove org as owner",
			token: ownerLoginSecret,
			id:    res.ID,
			err:   nil,
		},
		{
			desc:  "remove removed org",
			token: ownerLoginSecret,
			id:    res.ID,
			err:   errors.ErrNotFound,
		},
		{
			desc:  "remove non-existing org",
			token: ownerLoginSecret,
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

	_, ownerLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, viewerLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: viewerID, Subject: viewerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, editorLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: editorID, Subject: editorEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, adminLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: adminID, Subject: adminEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	res, err := svc.CreateOrg(context.Background(), ownerLoginSecret, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	err = svc.AssignMembers(context.Background(), ownerLoginSecret, res.ID, members...)
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
			token: viewerLoginSecret,
			org:   upOrg,
			err:   errors.ErrAuthorization,
		},
		{
			desc:  "update org as editor",
			token: editorLoginSecret,
			org:   upOrg,
			err:   errors.ErrAuthorization,
		},
		{
			desc:  "update org as admin",
			token: adminLoginSecret,
			org:   upOrg,
			err:   nil,
		},
		{
			desc:  "update org as owner",
			token: ownerLoginSecret,
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
			token: ownerLoginSecret,
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

	_, ownerLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, viewerLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: viewerID, Subject: viewerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, editorLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: editorID, Subject: editorEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, adminLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: adminID, Subject: adminEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	or, err := svc.CreateOrg(context.Background(), ownerLoginSecret, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	err = svc.AssignMembers(context.Background(), ownerLoginSecret, or.ID, members...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

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
			token: ownerLoginSecret,
			orgID: or.ID,
			org:   orgRes,
			err:   nil,
		},
		{
			desc:  "view org as viewer",
			token: viewerLoginSecret,
			orgID: or.ID,
			org:   orgRes,
			err:   nil,
		},
		{
			desc:  "view org for as editor",
			token: editorLoginSecret,
			orgID: or.ID,
			org:   orgRes,
			err:   nil,
		},
		{
			desc:  "view org as admin",
			token: adminLoginSecret,
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
			token: ownerLoginSecret,
			orgID: "",
			org:   auth.Org{},
			err:   errors.ErrNotFound,
		},
		{
			desc:  "view non-existing org",
			token: ownerLoginSecret,
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

	_, ownerLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, viewerLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: viewerID, Subject: viewerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, editorLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: editorID, Subject: editorEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, adminLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: adminID, Subject: adminEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	or, err := svc.CreateOrg(context.Background(), ownerLoginSecret, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	mb := []auth.Member{
		{
			ID:   "member1",
			Role: auth.ViewerRole,
		},
		{
			ID:   "member2",
			Role: auth.ViewerRole,
		},
	}
	cases := []struct {
		desc   string
		token  string
		orgID  string
		member []auth.Member
		err    error
	}{
		{
			desc:   "assign members to org as owner",
			token:  ownerLoginSecret,
			orgID:  or.ID,
			member: members,
			err:    nil,
		},
		{
			desc:   "assign members to org as admin",
			token:  adminLoginSecret,
			orgID:  or.ID,
			member: mb,
			err:    nil,
		},
		{
			desc:   "assign members to org as editor",
			token:  editorLoginSecret,
			orgID:  or.ID,
			member: mb,
			err:    errors.ErrAuthorization,
		},
		{
			desc:   "assign members to org as viewer",
			token:  viewerLoginSecret,
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
			token:  ownerLoginSecret,
			orgID:  invalid,
			member: members,
			err:    errors.ErrNotFound,
		},
		{
			desc:   "assign members to org without id",
			token:  ownerLoginSecret,
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

func TestUnAssignMembers(t *testing.T) {
	svc := newService()

	_, ownerLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, viewerLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: viewerID, Subject: viewerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, editorLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: editorID, Subject: editorEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, adminLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: adminID, Subject: adminEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	or, err := svc.CreateOrg(context.Background(), ownerLoginSecret, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	err = svc.AssignMembers(context.Background(), ownerLoginSecret, or.ID, members...)
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
			token:    viewerLoginSecret,
			orgID:    or.ID,
			memberID: editorID,
			err:      errors.ErrAuthorization,
		},
		{
			desc:     "unassign member from org as editor",
			token:    editorLoginSecret,
			orgID:    or.ID,
			memberID: viewerID,
			err:      errors.ErrAuthorization,
		},
		{
			desc:     "unassign member from org as admin",
			token:    adminLoginSecret,
			orgID:    or.ID,
			memberID: viewerID,
			err:      nil,
		},
		{
			desc:     "unassign owner from org as admin",
			token:    adminLoginSecret,
			orgID:    or.ID,
			memberID: id,
			err:      errors.ErrAuthorization,
		},
		{
			desc:     "unassign member from org as owner",
			token:    ownerLoginSecret,
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
			token:    ownerLoginSecret,
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

	_, ownerLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, viewerLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: viewerID, Subject: viewerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, editorLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: editorID, Subject: editorEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, adminLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: adminID, Subject: adminEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	or, err := svc.CreateOrg(context.Background(), ownerLoginSecret, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	err = svc.AssignMembers(context.Background(), ownerLoginSecret, or.ID, members...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := []struct {
		desc   string
		token  string
		orgID  string
		member auth.Member
		err    error
	}{
		{
			desc:   "update org member role as viewer",
			token:  viewerLoginSecret,
			orgID:  or.ID,
			member: members[1],
			err:    errors.ErrAuthorization,
		},
		{
			desc:   "update org member role as editor",
			token:  editorLoginSecret,
			orgID:  or.ID,
			member: members[2],
			err:    errors.ErrAuthorization,
		},
		{
			desc:   "update org member role as admin",
			token:  adminLoginSecret,
			orgID:  or.ID,
			member: members[2],
			err:    nil,
		},
		{
			desc:   "update org member role as owner",
			token:  ownerLoginSecret,
			orgID:  or.ID,
			member: members[1],
			err:    nil,
		},
		{
			desc:   "update org member role as owner",
			token:  ownerLoginSecret,
			orgID:  or.ID,
			member: members[1],
			err:    nil,
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
			token:  ownerLoginSecret,
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

func TestListOrgMembers(t *testing.T) {
	svc := newService()

	_, ownerLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, viewerLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: viewerID, Subject: viewerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, editorLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: editorID, Subject: editorEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, adminLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: adminID, Subject: adminEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	or, err := svc.CreateOrg(context.Background(), ownerLoginSecret, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	err = svc.AssignMembers(context.Background(), ownerLoginSecret, or.ID, members...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	var n uint64 = 4

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
			token: ownerLoginSecret,
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
			token: adminLoginSecret,
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
			token: editorLoginSecret,
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
			token: viewerLoginSecret,
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
			token: viewerLoginSecret,
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
			token: viewerLoginSecret,
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
			token: ownerLoginSecret,
			orgID: invalid,
			meta:  auth.PageMetadata{},
			size:  0,
			err:   nil,
		},
	}

	for _, tc := range cases {
		page, err := svc.ListOrgMembers(context.Background(), tc.token, tc.orgID, tc.meta)
		size := uint64(len(page.Members))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s expected %d got %d\n", tc.desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestAssignGroups(t *testing.T) {
	svc := newService()

	_, ownerLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, viewerLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: viewerID, Subject: viewerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, editorLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: editorID, Subject: editorEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, adminLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: adminID, Subject: adminEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	var grIDs []string
	for i := 0; i <= 10; i++ {
		groupID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
		grIDs = append(grIDs, groupID)
	}
	or, err := svc.CreateOrg(context.Background(), ownerLoginSecret, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	err = svc.AssignMembers(context.Background(), ownerLoginSecret, or.ID, members...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := []struct {
		desc     string
		token    string
		orgID    string
		groupIDs []string
		err      error
	}{
		{
			desc:     "Assign groups to org as owner",
			token:    ownerLoginSecret,
			orgID:    or.ID,
			groupIDs: grIDs,
			err:      nil,
		},
		{
			desc:     "Assign groups to org as admin",
			token:    adminLoginSecret,
			orgID:    or.ID,
			groupIDs: grIDs,
			err:      nil,
		},
		{
			desc:     "Assign groups to org as editor",
			token:    editorLoginSecret,
			orgID:    or.ID,
			groupIDs: grIDs,
			err:      nil,
		},
		{
			desc:     "Assign groups to org as viewer",
			token:    viewerLoginSecret,
			orgID:    or.ID,
			groupIDs: grIDs,
			err:      errors.ErrAuthorization,
		},
		{
			desc:     "Assign groups to org with invalid credentials",
			token:    invalid,
			orgID:    or.ID,
			groupIDs: grIDs,
			err:      errors.ErrAuthentication,
		},
		{
			desc:     "Assign groups to org without token",
			token:    "",
			orgID:    or.ID,
			groupIDs: grIDs,
			err:      errors.ErrAuthentication,
		},
		{
			desc:     "Assign groups to non-existing org",
			token:    ownerLoginSecret,
			orgID:    invalid,
			groupIDs: grIDs,
			err:      errors.ErrNotFound,
		},
		{
			desc:     "Assign groups to org without org ID",
			token:    ownerLoginSecret,
			orgID:    "",
			groupIDs: grIDs,
			err:      errors.ErrNotFound,
		},
		{
			desc:     "Assign groups to org without group IDs",
			token:    ownerLoginSecret,
			orgID:    or.ID,
			groupIDs: []string{},
			err:      nil,
		},
	}

	for _, tc := range cases {
		err := svc.AssignGroups(context.Background(), tc.token, tc.orgID, tc.groupIDs...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestUnAssignGroups(t *testing.T) {
	svc := newService()

	_, ownerLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, viewerLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: viewerID, Subject: viewerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, editorLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: editorID, Subject: editorEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, adminLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: adminID, Subject: adminEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	var grIDs []string
	for i := 0; i < 10; i++ {
		groupID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
		grIDs = append(grIDs, groupID)
	}

	or, err := svc.CreateOrg(context.Background(), ownerLoginSecret, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	err = svc.AssignMembers(context.Background(), ownerLoginSecret, or.ID, members...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	err = svc.AssignGroups(context.Background(), ownerLoginSecret, or.ID, grIDs...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := []struct {
		desc     string
		token    string
		orgID    string
		groupIDs []string
		err      error
	}{
		{
			desc:     "Unassign  groups from org as owner",
			token:    ownerLoginSecret,
			orgID:    or.ID,
			groupIDs: grIDs[0:2],
			err:      nil,
		},
		{
			desc:     "Unassign groups from org as admin",
			token:    adminLoginSecret,
			orgID:    or.ID,
			groupIDs: grIDs[3:5],
			err:      nil,
		},
		{
			desc:     "Unassign groups from org as editor",
			token:    editorLoginSecret,
			orgID:    or.ID,
			groupIDs: grIDs[6:8],
			err:      nil,
		},
		{
			desc:     "Unassign groups from org as viewer",
			token:    viewerLoginSecret,
			orgID:    or.ID,
			groupIDs: grIDs,
			err:      errors.ErrAuthorization,
		},
		{
			desc:     "Unassign groups from org with invalid credentials",
			token:    invalid,
			orgID:    or.ID,
			groupIDs: grIDs,
			err:      errors.ErrAuthentication,
		},
		{
			desc:     "Unassign groups from org without token",
			token:    "",
			orgID:    or.ID,
			groupIDs: grIDs,
			err:      errors.ErrAuthentication,
		},
		{
			desc:     "Unassign groups from non existing org",
			token:    ownerLoginSecret,
			orgID:    invalid,
			groupIDs: grIDs,
			err:      errors.ErrNotFound,
		},
		{
			desc:     "Unassign groups from org without org ID",
			token:    ownerLoginSecret,
			orgID:    "",
			groupIDs: grIDs,
			err:      errors.ErrNotFound,
		},
		{
			desc:     "Unassign groups from org without group IDs",
			token:    ownerLoginSecret,
			orgID:    or.ID,
			groupIDs: []string{},
			err:      nil,
		},
	}

	for _, tc := range cases {
		err := svc.UnassignGroups(context.Background(), tc.token, tc.orgID, tc.groupIDs...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestListOrgGroups(t *testing.T) {
	svc := newService()

	_, ownerLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, viewerLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: viewerID, Subject: viewerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, editorLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: editorID, Subject: editorEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, adminLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: adminID, Subject: adminEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	var grIDs []string
	gr := createGroups()
	for _, g := range gr {
		grIDs = append(grIDs, g.ID)
	}
	or, err := svc.CreateOrg(context.Background(), ownerLoginSecret, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	err = svc.AssignMembers(context.Background(), ownerLoginSecret, or.ID, members...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	err = svc.AssignGroups(context.Background(), ownerLoginSecret, or.ID, grIDs...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	cases := []struct {
		desc  string
		token string
		orgID string
		meta  auth.PageMetadata
		size  int
		err   error
	}{
		{
			desc:  "List org groups as owner",
			token: ownerLoginSecret,
			orgID: or.ID,
			meta: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		},
		{
			desc:  "List org groups as admin",
			token: adminLoginSecret,
			orgID: or.ID,
			meta: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		},
		{
			desc:  "List org groups as editor",
			token: editorLoginSecret,
			orgID: or.ID,
			meta: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		},
		{
			desc:  "List org groups as viewer",
			token: viewerLoginSecret,
			orgID: or.ID,
			meta: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		},
		{
			desc:  "List half org groups",
			token: viewerLoginSecret,
			orgID: or.ID,
			meta: auth.PageMetadata{
				Offset: 0,
				Limit:  n / 2,
			},
			size: n / 2,
			err:  nil,
		},
		{
			desc:  "List last five org groups",
			token: viewerLoginSecret,
			orgID: or.ID,
			meta: auth.PageMetadata{
				Offset: n - 5,
				Limit:  5,
			},
			size: 5,
			err:  nil,
		},
		{
			desc:  "List last org group",
			token: viewerLoginSecret,
			orgID: or.ID,
			meta: auth.PageMetadata{
				Offset: n - 1,
				Limit:  1,
			},
			size: 1,
			err:  nil,
		},
		{
			desc:  "List org groups with wrong credentials",
			token: invalid,
			orgID: or.ID,
			meta:  auth.PageMetadata{},
			size:  0,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "List org groups without credentials",
			token: "",
			orgID: or.ID,
			meta:  auth.PageMetadata{},
			size:  0,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "List groups from non-existing org",
			token: ownerLoginSecret,
			orgID: invalid,
			meta:  auth.PageMetadata{},
			size:  0,
			err:   nil,
		},
	}

	for _, tc := range cases {
		page, err := svc.ListOrgGroups(context.Background(), tc.token, tc.orgID, tc.meta)
		size := len(page.Groups)
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s expected %d got %d\n", tc.desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestListOrgMemberships(t *testing.T) {
	svc := newService()

	_, ownerLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: id, Subject: email})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, viewerLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: viewerID, Subject: viewerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, editorLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: editorID, Subject: editorEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))
	_, adminLoginSecret, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: adminID, Subject: adminEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	for i := 0; i < n; i++ {
		o := auth.Org{
			Name:        fmt.Sprintf("org-%d", i),
			Description: fmt.Sprintf("description-%d", i),
		}

		or, err := svc.CreateOrg(context.Background(), ownerLoginSecret, o)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
		or.Name = fmt.Sprintf("%s-%d", or.Name, i)

		err = svc.AssignMembers(context.Background(), ownerLoginSecret, or.ID, members...)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	}

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
			token:    ownerLoginSecret,
			memberID: id,
			meta: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		},
		{
			desc:     "list member organisations as admin",
			token:    adminLoginSecret,
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
			token:    editorLoginSecret,
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
			token:    viewerLoginSecret,
			memberID: viewerID,
			meta: auth.PageMetadata{
				Offset: 0,
				Limit:  n,
			},
			size: n,
			err:  nil,
		},
		{
			desc:     "list half of member organisations",
			token:    ownerLoginSecret,
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
			token:    ownerLoginSecret,
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
		page, err := svc.ListOrgMemberships(context.Background(), tc.token, tc.memberID, tc.meta)
		size := uint64(len(page.Orgs))
		assert.Equal(t, tc.size, int(size), fmt.Sprintf("%s expected %d got %d\n", tc.desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s expected %s got %s\n", tc.desc, tc.err, err))
	}
}
