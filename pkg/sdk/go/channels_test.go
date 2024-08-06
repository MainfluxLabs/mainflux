// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package sdk_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdk "github.com/MainfluxLabs/mainflux/pkg/sdk/go"
)

const (
	name    = "name"
	groupID = "1"
)

var (
	ch1          = sdk.Channel{Name: "test1"}
	ch2          = sdk.Channel{ID: "fe6b4e92-cc98-425e-b0aa-000000000001", Name: "test1"}
	ch3          = sdk.Channel{ID: "fe6b4e92-cc98-425e-b0aa-000000000002", Name: "test2"}
	chPrefix     = "fe6b4e92-cc98-425e-b0aa-"
	emptyChannel = sdk.Channel{GroupID: "1"}
)

func TestCreateChannel(t *testing.T) {
	svc := newThingsService()
	ts := newThingsServer(svc)
	defer ts.Close()

	chWrongExtID := sdk.Channel{GroupID: groupID, ID: "b0aa-000000000001", Name: "1", Metadata: metadata}

	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)

	cases := []struct {
		desc    string
		channel sdk.Channel
		token   string
		groupID string
		err     error
		empty   bool
	}{
		{
			desc:    "create new channel",
			channel: ch1,
			token:   token,
			groupID: groupID,
			err:     nil,
			empty:   false,
		},
		{
			desc:    "create new channel with empty token",
			channel: ch1,
			token:   "",
			groupID: groupID,
			err:     createError(sdk.ErrFailedCreation, http.StatusUnauthorized),
			empty:   true,
		},
		{
			desc:    "create new channel with invalid token",
			channel: ch1,
			token:   wrongValue,
			groupID: groupID,
			err:     createError(sdk.ErrFailedCreation, http.StatusUnauthorized),
			empty:   true,
		},
		{
			desc:    "create new empty channel",
			channel: emptyChannel,
			token:   token,
			groupID: groupID,
			err:     nil,
			empty:   false,
		},
		{
			desc:    "create a new channel with external UUID",
			channel: ch2,
			token:   token,
			groupID: groupID,
			err:     nil,
			empty:   false,
		},
		{
			desc:    "create a new channel with wrong external UUID",
			channel: chWrongExtID,
			token:   token,
			groupID: groupID,
			err:     createError(sdk.ErrFailedCreation, http.StatusBadRequest),
			empty:   true,
		},
	}

	for _, tc := range cases {
		loc, err := mainfluxSDK.CreateChannel(tc.channel, groupID, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.empty, loc == "", fmt.Sprintf("%s: expected empty result location, got: %s", tc.desc, loc))
	}
}

func TestCreateChannels(t *testing.T) {
	svc := newThingsService()
	ts := newThingsServer(svc)
	defer ts.Close()

	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)

	channels := []sdk.Channel{
		ch2,
		ch3,
	}

	cases := []struct {
		desc     string
		channels []sdk.Channel
		token    string
		err      error
		res      []sdk.Channel
	}{
		{
			desc:     "create new channels",
			channels: channels,
			token:    token,
			err:      nil,
			res:      channels,
		},
		{
			desc:     "create new channels with empty channels",
			channels: []sdk.Channel{},
			token:    token,
			err:      createError(sdk.ErrFailedCreation, http.StatusBadRequest),
			res:      []sdk.Channel{},
		},
		{
			desc:     "create new channels with empty token",
			channels: channels,
			token:    "",
			err:      createError(sdk.ErrFailedCreation, http.StatusUnauthorized),
			res:      []sdk.Channel{},
		},
		{
			desc:     "create new channels with invalid token",
			channels: channels,
			token:    wrongValue,
			err:      createError(sdk.ErrFailedCreation, http.StatusUnauthorized),
			res:      []sdk.Channel{},
		},
	}
	for _, tc := range cases {
		res, err := mainfluxSDK.CreateChannels(tc.channels, groupID, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))

		for idx := range tc.res {
			assert.Equal(t, tc.res[idx].ID, res[idx].ID, fmt.Sprintf("%s: expected response ID %s got %s", tc.desc, tc.res[idx].ID, res[idx].ID))
		}
	}
}

func TestChannel(t *testing.T) {
	svc := newThingsService()
	ts := newThingsServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)

	grID, err := mainfluxSDK.CreateGroup(group, orgID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	id, err := mainfluxSDK.CreateChannel(ch2, grID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	ch2.GroupID = grID

	cases := []struct {
		desc     string
		chanID   string
		token    string
		err      error
		response sdk.Channel
	}{
		{
			desc:     "get existing channel",
			chanID:   id,
			token:    token,
			err:      nil,
			response: ch2,
		},
		{
			desc:     "get non-existent channel",
			chanID:   "43",
			token:    token,
			err:      createError(sdk.ErrFailedFetch, http.StatusNotFound),
			response: sdk.Channel{},
		},
		{
			desc:     "get channel with invalid token",
			chanID:   id,
			token:    "",
			err:      createError(sdk.ErrFailedFetch, http.StatusUnauthorized),
			response: sdk.Channel{},
		},
	}

	for _, tc := range cases {
		respCh, err := mainfluxSDK.Channel(tc.chanID, tc.token)

		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, respCh, fmt.Sprintf("%s: expected response channel %s, got %s", tc.desc, tc.response, respCh))
	}
}

func TestChannels(t *testing.T) {
	svc := newThingsService()
	ts := newThingsServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}
	var channels []sdk.Channel
	mainfluxSDK := sdk.NewSDK(sdkConf)
	for i := 1; i < 101; i++ {
		id := fmt.Sprintf("%s%012d", chPrefix, i)
		name := fmt.Sprintf("test-%d", i)
		ch := sdk.Channel{GroupID: "1", ID: id, Name: name}
		_, err := mainfluxSDK.CreateChannel(ch, groupID, token)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		channels = append(channels, ch)
	}

	cases := []struct {
		desc     string
		token    string
		offset   uint64
		limit    uint64
		name     string
		err      error
		response []sdk.Channel
		metadata map[string]interface{}
	}{
		{
			desc:     "get a list of channels",
			token:    token,
			offset:   offset,
			limit:    limit,
			err:      nil,
			response: channels[0:limit],
			metadata: make(map[string]interface{}),
		},
		{
			desc:     "get a list of channels with invalid token",
			token:    wrongValue,
			offset:   offset,
			limit:    limit,
			err:      createError(sdk.ErrFailedFetch, http.StatusUnauthorized),
			response: nil,
			metadata: make(map[string]interface{}),
		},
		{
			desc:     "get a list of channels with empty token",
			token:    "",
			offset:   offset,
			limit:    limit,
			err:      createError(sdk.ErrFailedFetch, http.StatusUnauthorized),
			response: nil,
			metadata: make(map[string]interface{}),
		},
		{
			desc:     "get a list of channels without limit, default 10",
			token:    token,
			offset:   offset,
			limit:    0,
			err:      createError(sdk.ErrFailedFetch, http.StatusBadRequest),
			response: nil,
			metadata: make(map[string]interface{}),
		},
		{
			desc:     "get a list of channels with limit greater than max",
			token:    token,
			offset:   offset,
			limit:    110,
			err:      createError(sdk.ErrFailedFetch, http.StatusBadRequest),
			response: nil,
			metadata: make(map[string]interface{}),
		},
		{
			desc:     "get a list of channels with offset greater than max",
			token:    token,
			offset:   110,
			limit:    limit,
			err:      nil,
			response: []sdk.Channel{},
			metadata: make(map[string]interface{}),
		},
	}
	for _, tc := range cases {
		filter := sdk.PageMetadata{
			Name:     tc.name,
			Total:    total,
			Offset:   uint64(tc.offset),
			Limit:    uint64(tc.limit),
			Metadata: tc.metadata,
		}

		page, err := mainfluxSDK.Channels(tc.token, filter)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, page.Channels, fmt.Sprintf("%s: expected response channel %s, got %s", tc.desc, tc.response, page.Channels))
	}
}

func TestViewChannelByThing(t *testing.T) {
	svc := newThingsService()
	ts := newThingsServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}
	mainfluxSDK := sdk.NewSDK(sdkConf)

	grID, err := mainfluxSDK.CreateGroup(group, orgID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	th := sdk.Thing{Name: name}
	tid, err := mainfluxSDK.CreateThing(th, grID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	ch := sdk.Channel{Name: name}
	cid, err := mainfluxSDK.CreateChannel(ch, grID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	connIDs := sdk.ConnectionIDs{
		ChannelID: cid,
		ThingIDs:  []string{tid},
	}

	resChan := sdk.Channel{
		ID:      cid,
		GroupID: grID,
		Name:    name,
	}

	err = mainfluxSDK.Connect(connIDs, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc     string
		thing    string
		token    string
		err      error
		response sdk.Channel
	}{
		{
			desc:     "view channel by thing",
			thing:    tid,
			token:    token,
			err:      nil,
			response: resChan,
		},
		{
			desc:     "view channel by thing with invalid token",
			thing:    tid,
			token:    wrongValue,
			err:      createError(sdk.ErrFailedFetch, http.StatusUnauthorized),
			response: sdk.Channel{},
		},
		{
			desc:     "view channel by thing with empty token",
			thing:    tid,
			token:    "",
			err:      createError(sdk.ErrFailedFetch, http.StatusUnauthorized),
			response: sdk.Channel{},
		},
		{
			desc:     "view channel by thing with empty thing id",
			thing:    "",
			token:    token,
			err:      createError(sdk.ErrFailedFetch, http.StatusBadRequest),
			response: sdk.Channel{},
		},
		{
			desc:     "view channel by thing with unknown thing id",
			thing:    wrongID,
			token:    token,
			err:      createError(sdk.ErrFailedFetch, http.StatusNotFound),
			response: sdk.Channel{},
		},
	}

	for _, tc := range cases {
		ch, err := mainfluxSDK.ViewChannelByThing(tc.token, tc.thing)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
		assert.Equal(t, tc.response, ch, fmt.Sprintf("%s: expected response channel %s, got %s", tc.desc, tc.response, ch))
	}
}

func TestUpdateChannel(t *testing.T) {
	svc := newThingsService()
	ts := newThingsServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)

	grID, err := mainfluxSDK.CreateGroup(group, orgID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	id, err := mainfluxSDK.CreateChannel(ch2, grID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error %s", err))
	ch2.GroupID = grID

	cases := []struct {
		desc    string
		channel sdk.Channel
		token   string
		err     error
	}{
		{
			desc:    "update existing channel",
			channel: sdk.Channel{ID: id, Name: "test2"},
			token:   token,
			err:     nil,
		},
		{
			desc:    "update non-existing channel",
			channel: sdk.Channel{ID: "0", Name: "test2"},
			token:   token,
			err:     createError(sdk.ErrFailedUpdate, http.StatusNotFound),
		},
		{
			desc:    "update channel with invalid id",
			channel: sdk.Channel{ID: "", Name: "test2"},
			token:   token,
			err:     createError(sdk.ErrFailedUpdate, http.StatusBadRequest),
		},
		{
			desc:    "update channel with invalid token",
			channel: sdk.Channel{ID: id, Name: "test2"},
			token:   wrongValue,
			err:     createError(sdk.ErrFailedUpdate, http.StatusUnauthorized),
		},
		{
			desc:    "update channel with empty token",
			channel: sdk.Channel{ID: id, Name: "test2"},
			token:   "",
			err:     createError(sdk.ErrFailedUpdate, http.StatusUnauthorized),
		},
	}

	for _, tc := range cases {
		err := mainfluxSDK.UpdateChannel(tc.channel, tc.channel.ID, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}

func TestDeleteChannel(t *testing.T) {
	svc := newThingsService()
	ts := newThingsServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)
	id, err := mainfluxSDK.CreateChannel(ch2, groupID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc   string
		chanID string
		token  string
		err    error
	}{
		{
			desc:   "delete channel with invalid token",
			chanID: id,
			token:  wrongValue,
			err:    createError(sdk.ErrFailedRemoval, http.StatusUnauthorized),
		},
		{
			desc:   "delete non-existing channel",
			chanID: "2",
			token:  token,
			err:    createError(sdk.ErrFailedRemoval, http.StatusNotFound),
		},
		{
			desc:   "delete channel with invalid id",
			chanID: "",
			token:  token,
			err:    createError(sdk.ErrFailedRemoval, http.StatusBadRequest),
		},
		{
			desc:   "delete channel with empty token",
			chanID: id,
			token:  "",
			err:    createError(sdk.ErrFailedRemoval, http.StatusUnauthorized),
		},
		{
			desc:   "delete existing channel",
			chanID: id,
			token:  token,
			err:    nil,
		},
		{
			desc:   "delete deleted channel",
			chanID: id,
			token:  token,
			err:    createError(sdk.ErrFailedRemoval, http.StatusNotFound),
		},
	}

	for _, tc := range cases {
		err := mainfluxSDK.DeleteChannel(tc.chanID, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}

func TestDeleteChannels(t *testing.T) {
	svc := newThingsService()
	ts := newThingsServer(svc)
	defer ts.Close()
	sdkConf := sdk.Config{
		ThingsURL:       ts.URL,
		MsgContentType:  contentType,
		TLSVerification: false,
	}

	mainfluxSDK := sdk.NewSDK(sdkConf)
	id1, err := mainfluxSDK.CreateChannel(ch1, groupID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	id2, err := mainfluxSDK.CreateChannel(ch2, groupID, token)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	chIDs := []string{id1, id2}

	cases := []struct {
		desc    string
		chanIDs []string
		token   string
		err     error
	}{
		{
			desc:    "delete channels with invalid token",
			chanIDs: chIDs,
			token:   wrongValue,
			err:     createError(sdk.ErrFailedRemoval, http.StatusUnauthorized),
		},
		{
			desc:    "delete non-existing channels",
			chanIDs: []string{wrongValue},
			token:   token,
			err:     createError(sdk.ErrFailedRemoval, http.StatusNotFound),
		},
		{
			desc:    "delete channels with empty id",
			chanIDs: []string{""},
			token:   token,
			err:     createError(sdk.ErrFailedRemoval, http.StatusBadRequest),
		},
		{
			desc:    "delete channels without channel ids",
			chanIDs: []string{},
			token:   token,
			err:     createError(sdk.ErrFailedRemoval, http.StatusBadRequest),
		},
		{
			desc:    "delete channels with empty token",
			chanIDs: chIDs,
			token:   "",
			err:     createError(sdk.ErrFailedRemoval, http.StatusUnauthorized),
		},
		{
			desc:    "delete channels with invalid token",
			chanIDs: chIDs,
			token:   wrongValue,
			err:     createError(sdk.ErrFailedRemoval, http.StatusUnauthorized),
		},
		{
			desc:    "delete existing channels",
			chanIDs: chIDs,
			token:   token,
			err:     nil,
		},
		{
			desc:    "delete deleted channels",
			chanIDs: chIDs,
			token:   token,
			err:     createError(sdk.ErrFailedRemoval, http.StatusNotFound),
		},
	}

	for _, tc := range cases {
		err := mainfluxSDK.DeleteChannels(tc.chanIDs, tc.token)
		assert.Equal(t, tc.err, err, fmt.Sprintf("%s: expected error %s, got %s", tc.desc, tc.err, err))
	}
}
