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
	"google.golang.org/grpc/status"
)

const wrongID = ""

var (
	thing   = things.Thing{Name: "test_app", Metadata: map[string]interface{}{"test": "test"}}
	channel = things.Channel{Name: "test", Metadata: map[string]interface{}{"test": "test", "profile": things.Profile{ContentType: "application/json"}}}
	group   = things.Group{Name: "test-group", Description: "test-group-desc"}
)

func TestGetConnByKey(t *testing.T) {
	grs, err := svc.CreateGroups(context.Background(), token, group)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	gr := grs[0]

	thing.GroupID = gr.ID
	ths, err := svc.CreateThings(context.Background(), token, thing, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	th1 := ths[0]
	th2 := ths[1]

	channel.GroupID = gr.ID
	chs, err := svc.CreateChannels(context.Background(), token, channel)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch := chs[0]

	err = svc.Connect(context.Background(), token, ch.ID, []string{th1.ID})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	usersAddr := fmt.Sprintf("localhost:%d", port)
	conn, err := grpc.Dial(usersAddr, grpc.WithInsecure())
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	cli := grpcapi.NewClient(conn, mocktracer.New(), time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cases := map[string]struct {
		key  string
		code codes.Code
	}{
		"check if connected thing can access existing channel": {
			key:  th1.Key,
			code: codes.OK,
		},
		"check if unconnected thing can access existing channel": {
			key:  th2.Key,
			code: codes.PermissionDenied,
		},
		"check if thing with wrong access key can access existing channel": {
			key:  wrong,
			code: codes.NotFound,
		},
	}

	for desc, tc := range cases {
		_, err := cli.GetConnByKey(ctx, &protomfx.ConnByKeyReq{Key: tc.key})
		e, ok := status.FromError(err)
		assert.True(t, ok, "OK expected to be true")
		assert.Equal(t, tc.code, e.Code(), fmt.Sprintf("%s: expected %s got %s", desc, tc.code, e.Code()))
	}
}

func TestIdentify(t *testing.T) {
	ths, err := svc.CreateThings(context.Background(), token, thing)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	sth := ths[0]

	usersAddr := fmt.Sprintf("localhost:%d", port)
	conn, err := grpc.Dial(usersAddr, grpc.WithInsecure())
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	cli := grpcapi.NewClient(conn, mocktracer.New(), time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	cases := map[string]struct {
		key  string
		id   string
		code codes.Code
	}{
		"identify existing thing": {
			key:  sth.Key,
			id:   sth.ID,
			code: codes.OK,
		},
		"identify non-existent thing": {
			key:  wrong,
			id:   wrongID,
			code: codes.NotFound,
		},
	}

	for desc, tc := range cases {
		id, err := cli.Identify(ctx, &protomfx.Token{Value: tc.key})
		e, ok := status.FromError(err)
		assert.True(t, ok, "OK expected to be true")
		assert.Equal(t, tc.id, id.GetValue(), fmt.Sprintf("%s: expected %s got %s", desc, tc.id, id.GetValue()))
		assert.Equal(t, tc.code, e.Code(), fmt.Sprintf("%s: expected %s got %s", desc, tc.code, e.Code()))
	}
}
