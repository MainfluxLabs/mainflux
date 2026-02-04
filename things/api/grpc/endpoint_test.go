// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package grpc_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
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
		_, err := cli.GetPubConfigByKey(ctx, &protomfx.ThingKey{Value: tc.key, Type: things.KeyTypeInternal})
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
		id, err := cli.Identify(ctx, &protomfx.ThingKey{Value: tc.key, Type: tc.keyType})
		e, ok := status.FromError(err)
		assert.True(t, ok, "OK expected to be true")
		assert.Equal(t, tc.id, id.GetValue(), fmt.Sprintf("%s: expected %s got %s", desc, tc.id, id.GetValue()))
		assert.Equal(t, tc.code, e.Code(), fmt.Sprintf("%s: expected %s got %s", desc, tc.code, e.Code()))
	}
}
