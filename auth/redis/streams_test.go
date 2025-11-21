// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis_test

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/auth/jwt"
	"github.com/MainfluxLabs/mainflux/auth/mocks"
	"github.com/MainfluxLabs/mainflux/auth/redis"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	thmocks "github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/users"
	r "github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	streamID = "mainflux.auth"

	secret      = "secret"
	ownerEmail  = "owner@test.com"
	ownerID     = "ownerID"
	description = "description"
	name        = "name"
	n           = 10

	loginDuration  = 30 * time.Minute
	inviteDuration = 7 * 24 * time.Hour

	orgPrefix = "org."
	orgCreate = orgPrefix + "create"
	orgRemove = orgPrefix + "remove"
)

var (
	org           = auth.Org{Name: name, Description: description, Metadata: map[string]interface{}{"test": "test"}}
	usersByEmails = map[string]users.User{ownerEmail: {ID: ownerID, Email: ownerEmail}}
	usersByIDs    = map[string]users.User{ownerID: {ID: ownerID, Email: ownerEmail}}
)

func newService() auth.Service {
	keyRepo := mocks.NewKeyRepository()
	idMockProvider := uuid.NewMock()
	membsRepo := mocks.NewOrgMembershipsRepository()
	orgRepo := mocks.NewOrgRepository(membsRepo)
	roleRepo := mocks.NewRolesRepository()
	invitesRepo := mocks.NewInvitesRepository()
	emailerMock := mocks.NewEmailer()

	uc := mocks.NewUsersService(usersByIDs, usersByEmails)
	tc := thmocks.NewThingsServiceClient(nil, nil, nil)
	t := jwt.New(secret)

	return auth.New(orgRepo, tc, uc, keyRepo, roleRepo, membsRepo, invitesRepo, emailerMock, idMockProvider, t, loginDuration, inviteDuration)
}

func TestCreateOrg(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService()

	_, ownerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: ownerID, Subject: ownerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	svc = redis.NewEventStoreMiddleware(svc, redisClient)

	cases := []struct {
		desc  string
		token string
		org   auth.Org
		err   error
		event map[string]interface{}
	}{
		{
			desc:  "create org successfully",
			org:   org,
			token: ownerToken,
			err:   nil,
			event: map[string]interface{}{
				"id":        "123e4567-e89b-12d3-a456-000000000001",
				"operation": orgCreate,
			},
		},
		{
			desc:  "create org with invalid credentials",
			org:   org,
			token: "invalid",
			err:   errors.ErrAuthentication,
			event: nil,
		},
	}

	lastID := "0"
	for _, oc := range cases {
		_, err := svc.CreateOrg(context.Background(), oc.token, oc.org)
		assert.True(t, errors.Contains(err, oc.err), fmt.Sprintf("%s: expected %s got %s\n", oc.desc, oc.err, err))

		streams := redisClient.XRead(context.Background(), &r.XReadArgs{
			Streams: []string{streamID, lastID},
			Count:   1,
			Block:   time.Second,
		}).Val()

		var event map[string]interface{}
		if len(streams) > 0 && len(streams[0].Messages) > 0 {
			msg := streams[0].Messages[0]
			event = msg.Values
			lastID = msg.ID
		}

		assert.Equal(t, oc.event, event, fmt.Sprintf("%s: expected %v got %v\n", oc.desc, oc.event, event))
	}
}

func TestRemoveOrg(t *testing.T) {
	_ = redisClient.FlushAll(context.Background()).Err()

	svc := newService()

	_, ownerToken, err := svc.Issue(context.Background(), "", auth.Key{Type: auth.LoginKey, IssuedAt: time.Now(), IssuerID: ownerID, Subject: ownerEmail})
	assert.Nil(t, err, fmt.Sprintf("Issuing login key expected to succeed: %s", err))

	org, err := svc.CreateOrg(context.Background(), ownerToken, org)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	svc = redis.NewEventStoreMiddleware(svc, redisClient)

	cases := []struct {
		desc  string
		id    string
		token string
		err   error
		event map[string]interface{}
	}{
		{
			desc:  "remove existing org successfully",
			id:    org.ID,
			token: ownerToken,
			err:   nil,
			event: map[string]interface{}{
				"id":        org.ID,
				"operation": orgRemove,
			},
		},
		{
			desc:  "remove non-existent org",
			id:    strconv.FormatUint(math.MaxUint64, 10),
			token: ownerToken,
			err:   dbutil.ErrNotFound,
			event: nil,
		},
	}

	lastID := "0"
	for _, oc := range cases {
		err := svc.RemoveOrgs(context.Background(), oc.token, oc.id)
		assert.True(t, errors.Contains(err, oc.err), fmt.Sprintf("%s: expected %s got %s\n", oc.desc, oc.err, err))

		streams := redisClient.XRead(context.Background(), &r.XReadArgs{
			Streams: []string{streamID, lastID},
			Count:   1,
			Block:   time.Second,
		}).Val()

		var event map[string]interface{}
		if len(streams) > 0 && len(streams[0].Messages) > 0 {
			msg := streams[0].Messages[0]
			event = msg.Values
			lastID = msg.ID
		}

		assert.Equal(t, oc.event, event, fmt.Sprintf("%s: expected %v got %v\n", oc.desc, oc.event, event))
	}
}
