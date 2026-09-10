// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/things"
	grpcapi "github.com/MainfluxLabs/mainflux/things/api/grpc"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

const wrongID = ""

var (
	thing   = things.Thing{Name: "test_app", Metadata: map[string]any{"test": "test"}}
	profile = things.Profile{Name: "test", Metadata: map[string]any{"test": "test", "config": things.Config{ContentType: "application/json"}}}
	group   = things.Group{Name: "test-group", Description: "test-group-desc"}
)

func TestGetPubConfigByKey(t *testing.T) {
	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	prID := prs[0].ID

	thing.GroupID = grID
	ths, err := svc.CreateThings(context.Background(), token, prID, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	thKey := ths[0].Key

	usersAddr := fmt.Sprintf("localhost:%d", port)
	conn, err := grpc.NewClient(usersAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	cli := grpcapi.NewClient(conn, mocktracer.New(), time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cases := map[string]struct {
		key  string
		code codes.Code
	}{
		"check if thing can access existing profile": {
			key:  thKey,
			code: codes.OK,
		},
		"check if thing with wrong access key can access existing profile": {
			key:  wrong,
			code: codes.NotFound,
		},
	}

	for desc, tc := range cases {
		_, err := cli.GetPubConfigByKey(ctx, domain.ThingKey{Value: tc.key, Type: things.KeyTypeInternal})
		e, ok := status.FromError(err)
		assert.True(t, ok, "OK expected to be true")
		assert.Equal(t, tc.code, e.Code(), fmt.Sprintf("%s: expected %s got %s", desc, tc.code, e.Code()))
	}
}

func TestIdentify(t *testing.T) {
	grs, err := svc.CreateGroups(context.Background(), token, orgID, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, profile)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	prID := prs[0].ID

	thing.GroupID = grID
	ths, err := svc.CreateThings(context.Background(), token, prID, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	sth := ths[0]

	externalKey := "abc123"
	err = svc.UpdateExternalKey(context.Background(), token, externalKey, sth.ID)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	usersAddr := fmt.Sprintf("localhost:%d", port)
	conn, err := grpc.NewClient(usersAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	cli := grpcapi.NewClient(conn, mocktracer.New(), time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cases := map[string]struct {
		key     string
		keyType string
		id      string
		code    codes.Code
	}{
		"identify thing using internal key": {
			key:     sth.Key,
			keyType: things.KeyTypeInternal,
			id:      sth.ID,
			code:    codes.OK,
		},
		"identify thing using invalid internal key": {
			key:     wrong,
			keyType: things.KeyTypeInternal,
			id:      wrongID,
			code:    codes.NotFound,
		},
		"identify thing using external key": {
			key:     externalKey,
			keyType: things.KeyTypeExternal,
			id:      sth.ID,
			code:    codes.OK,
		},
		"identify thing using invalid external key": {
			key:     wrong,
			keyType: things.KeyTypeExternal,
			id:      wrongID,
			code:    codes.NotFound,
		},
	}

	for desc, tc := range cases {
		id, err := cli.Identify(ctx, domain.ThingKey{Value: tc.key, Type: tc.keyType})
		e, ok := status.FromError(err)
		assert.True(t, ok, "OK expected to be true")
		assert.Equal(t, tc.id, id, fmt.Sprintf("%s: expected %s got %s", desc, tc.id, id))
		assert.Equal(t, tc.code, e.Code(), fmt.Sprintf("%s: expected %s got %s", desc, tc.code, e.Code()))
	}
}

func TestCanUserAccessThings(t *testing.T) {
	grs, err := svc.CreateGroups(context.Background(), token, orgID, things.Group{Name: "batch-access-group", Description: "batch-access-group-desc"})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	grID := grs[0].ID

	prs, err := svc.CreateProfiles(context.Background(), token, grID, things.Profile{Name: "batch-access-profile"})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	prID := prs[0].ID

	ths, err := svc.CreateThings(context.Background(), token, prID, things.Thing{Name: "batch-1"}, things.Thing{Name: "batch-2"})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	thIDs := []string{ths[0].ID, ths[1].ID}

	otherGrs, err := svc.CreateGroups(context.Background(), token, orgID, things.Group{Name: "other-batch-access-group"})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	otherGrID := otherGrs[0].ID

	usersAddr := fmt.Sprintf("localhost:%d", port)
	conn, err := grpc.NewClient(usersAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	cli := grpcapi.NewClient(conn, mocktracer.New(), time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cases := map[string]struct {
		token   string
		ids     []string
		groupID string
		action  string
		code    codes.Code
	}{
		"check user access to things": {
			token:  token,
			ids:    thIDs,
			action: things.Viewer,
			code:   codes.OK,
		},
		"check user access to things with matching group id": {
			token:   token,
			ids:     thIDs,
			groupID: grID,
			action:  things.Viewer,
			code:    codes.OK,
		},
		"check user access to things with a group id they do not belong to": {
			token:   token,
			ids:     thIDs,
			groupID: otherGrID,
			action:  things.Viewer,
			code:    codes.PermissionDenied,
		},
		"check user access to things with a non-existing thing id": {
			token:  token,
			ids:    append([]string{wrong}, thIDs...),
			action: things.Viewer,
			code:   codes.NotFound,
		},
		"check user access to things with an empty list of ids": {
			token:  token,
			ids:    []string{},
			action: things.Viewer,
			code:   codes.InvalidArgument,
		},
		"check user access to things with an empty thing id": {
			token:  token,
			ids:    []string{""},
			action: things.Viewer,
			code:   codes.InvalidArgument,
		},
		"check user access to things with an invalid action": {
			token:  token,
			ids:    thIDs,
			action: wrong,
			code:   codes.InvalidArgument,
		},
		"check user access to things with an empty token": {
			token:  "",
			ids:    thIDs,
			action: things.Viewer,
			code:   codes.InvalidArgument,
		},
	}

	for desc, tc := range cases {
		err := cli.CanUserAccessThings(ctx, domain.UserAccessThingsReq{Token: tc.token, IDs: tc.ids, GroupID: tc.groupID, Action: tc.action})
		e, ok := status.FromError(err)
		assert.True(t, ok, "OK expected to be true")
		assert.Equal(t, tc.code, e.Code(), fmt.Sprintf("%s: expected %s got %s", desc, tc.code, e.Code()))
	}
}
