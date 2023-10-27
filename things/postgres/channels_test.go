// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/MainfluxLabs/mainflux/things/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChannelsSave(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	channelRepo := postgres.NewChannelRepository(dbMiddleware)

	email := "channel-save@example.com"

	chs := []things.Channel{}
	for i := 1; i <= 5; i++ {
		id, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		ch := things.Channel{
			ID:    id,
			Owner: email,
		}
		chs = append(chs, ch)
	}
	id := chs[0].ID

	cases := []struct {
		desc     string
		channels []things.Channel
		err      error
	}{
		{
			desc:     "create new channels",
			channels: chs,
			err:      nil,
		},
		{
			desc:     "create channels that already exist",
			channels: chs,
			err:      errors.ErrConflict,
		},
		{
			desc: "create channel with invalid ID",
			channels: []things.Channel{
				{ID: "invalid", Owner: email},
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "create channel with invalid name",
			channels: []things.Channel{
				{ID: id, Owner: email, Name: invalidName},
			},
			err: errors.ErrMalformedEntity,
		},
		{
			desc: "create channel with invalid name",
			channels: []things.Channel{
				{ID: id, Owner: email, Name: invalidName},
			},
			err: errors.ErrMalformedEntity,
		},
	}

	for _, cc := range cases {
		_, err := channelRepo.Save(context.Background(), cc.channels...)
		assert.True(t, errors.Contains(err, cc.err), fmt.Sprintf("%s: expected %s got %s\n", cc.desc, cc.err, err))
	}
}

func TestChannelUpdate(t *testing.T) {
	email := "channel-update@example.com"
	dbMiddleware := postgres.NewDatabase(db)
	chanRepo := postgres.NewChannelRepository(dbMiddleware)

	id, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	ch := things.Channel{
		ID:    id,
		Owner: email,
	}

	chs, err := chanRepo.Save(context.Background(), ch)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch.ID = chs[0].ID

	nonexistentChanID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc    string
		channel things.Channel
		err     error
	}{
		{
			desc:    "update existing channel",
			channel: ch,
			err:     nil,
		},
		{
			desc: "update non-existing channel with existing user",
			channel: things.Channel{
				ID:    nonexistentChanID,
				Owner: email,
			},
			err: errors.ErrNotFound,
		},
		{
			desc: "update existing channel ID with non-existing user",
			channel: things.Channel{
				ID:    ch.ID,
				Owner: wrongValue,
			},
			err: errors.ErrNotFound,
		},
		{
			desc: "update non-existing channel with non-existing user",
			channel: things.Channel{
				ID:    nonexistentChanID,
				Owner: wrongValue,
			},
			err: errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := chanRepo.Update(context.Background(), tc.channel)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestSingleChannelRetrieval(t *testing.T) {
	email := "channel-single-retrieval@example.com"
	dbMiddleware := postgres.NewDatabase(db)
	chanRepo := postgres.NewChannelRepository(dbMiddleware)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	thID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thkey, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	th := things.Thing{
		ID:    thID,
		Owner: email,
		Key:   thkey,
	}
	ths, err := thingRepo.Save(context.Background(), th)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	th = ths[0]

	chID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	c := things.Channel{
		ID:    chID,
		Owner: email,
	}
	chs, err := chanRepo.Save(context.Background(), c)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	ch := chs[0]

	err = chanRepo.Connect(context.Background(), email, ch.ID, []string{th.ID})
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	nonexistentChanID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := map[string]struct {
		ID  string
		err error
	}{
		"retrieve channel with existing user": {
			ID:  ch.ID,
			err: nil,
		},
		"retrieve channel with existing user, non-existing channel": {
			ID:  nonexistentChanID,
			err: errors.ErrNotFound,
		},
		"retrieve channel with malformed ID": {
			ID:  wrongValue,
			err: errors.ErrNotFound,
		},
	}

	for desc, tc := range cases {
		_, err := chanRepo.RetrieveByID(context.Background(), tc.ID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestMultiChannelRetrieval(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	chanRepo := postgres.NewChannelRepository(dbMiddleware)

	email := "channel-multi-retrieval@example.com"
	name := "channel_name"
	metadata := things.Metadata{
		"field": "value",
	}
	wrongMeta := things.Metadata{
		"wrong": "wrong",
	}

	offset := uint64(1)
	nameNum := uint64(3)
	metaNum := uint64(3)
	nameMetaNum := uint64(2)

	n := uint64(101)
	for i := uint64(0); i < n; i++ {
		chID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		ch := things.Channel{
			ID:    chID,
			Owner: email,
		}

		// Create Channels with name.
		if i < nameNum {
			ch.Name = fmt.Sprintf("%s-%d", name, i)
		}
		// Create Channels with metadata.
		if i >= nameNum && i < nameNum+metaNum {
			ch.Metadata = metadata
		}
		// Create Channels with name and metadata.
		if i >= n-nameMetaNum {
			ch.Metadata = metadata
			ch.Name = name
		}

		chanRepo.Save(context.Background(), ch)
	}

	cases := map[string]struct {
		owner        string
		size         uint64
		pageMetadata things.PageMetadata
	}{
		"retrieve all channels with existing owner": {
			owner: email,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  n,
			},
			size: n,
		},
		"retrieve all channels with no limit for existing owner": {
			owner: email,
			pageMetadata: things.PageMetadata{
				Limit: 0,
				Total: n,
			},
			size: n,
		},
		"retrieve subset of channels with existing owner": {
			owner: email,
			pageMetadata: things.PageMetadata{
				Offset: offset,
				Limit:  n,
				Total:  n,
			},
			size: n - offset,
		},
		"retrieve channels with non-existing owner": {
			owner: wrongValue,
			pageMetadata: things.PageMetadata{
				Offset: n / 2,
				Limit:  n,
				Total:  0,
			},
			size: 0,
		},
		"retrieve channels with existing name": {
			owner: email,
			pageMetadata: things.PageMetadata{
				Offset: offset,
				Limit:  n,
				Name:   name,
				Total:  nameNum + nameMetaNum,
			},
			size: nameNum + nameMetaNum - offset,
		},
		"retrieve all channels with non-existing name": {
			owner: email,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
				Name:   "wrong",
				Total:  0,
			},
			size: 0,
		},
		"retrieve all channels with existing metadata": {
			owner: email,
			pageMetadata: things.PageMetadata{
				Offset:   0,
				Limit:    n,
				Metadata: metadata,
				Total:    metaNum + nameMetaNum,
			},
			size: metaNum + nameMetaNum,
		},
		"retrieve all channels with non-existing metadata": {
			owner: email,
			pageMetadata: things.PageMetadata{
				Offset:   0,
				Limit:    n,
				Metadata: wrongMeta,
				Total:    0,
			},
		},
		"retrieve all channels with existing name and metadata": {
			owner: email,
			pageMetadata: things.PageMetadata{
				Offset:   0,
				Limit:    n,
				Name:     name,
				Metadata: metadata,
				Total:    nameMetaNum,
			},
			size: nameMetaNum,
		},
		"retrieve channels sorted by name ascendent": {
			owner: email,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  n,
				Order:  "name",
				Dir:    "asc",
			},
			size: n,
		},
		"retrieve channels sorted by name descendent": {
			owner: email,
			pageMetadata: things.PageMetadata{
				Offset: 0,
				Limit:  n,
				Total:  n,
				Order:  "name",
				Dir:    "desc",
			},
			size: n,
		},
	}

	for desc, tc := range cases {
		page, err := chanRepo.RetrieveByOwner(context.Background(), tc.owner, tc.pageMetadata)
		size := uint64(len(page.Channels))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", desc, tc.size, size))
		assert.Equal(t, tc.pageMetadata.Total, page.Total, fmt.Sprintf("%s: expected total %d got %d\n", desc, tc.pageMetadata.Total, page.Total))
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %d\n", desc, err))

		// Check if Channels list have been sorted properly
		testSortChannels(t, tc.pageMetadata, page.Channels)
	}
}

func TestRetrieveByThing(t *testing.T) {
	email := "channel-multi-retrieval-by-thing@example.com"
	dbMiddleware := postgres.NewDatabase(db)
	chanRepo := postgres.NewChannelRepository(dbMiddleware)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	thID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	th, err := thingRepo.Save(context.Background(), things.Thing{
		ID:    thID,
		Owner: email,
	})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	thID = th[0].ID

	chID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	ch := things.Channel{
		ID:       chID,
		Owner:    email,
		Metadata: things.Metadata{},
	}

	_, err = chanRepo.Save(context.Background(), ch)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	err = chanRepo.Connect(context.Background(), email, chID, []string{thID})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	nonexistentThingID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := map[string]struct {
		owner   string
		thID    string
		channel things.Channel
		err     error
	}{
		"retrieve channel by thing with existing owner": {
			owner:   email,
			thID:    thID,
			channel: ch,
			err:     nil,
		},
		"retrieve channel by thing with non-existing owner": {
			owner:   wrongValue,
			thID:    thID,
			channel: things.Channel{},
			err:     nil,
		},
		"retrieve channel by non-existent thing": {
			owner:   email,
			thID:    nonexistentThingID,
			channel: things.Channel{},
			err:     nil,
		},
		"retrieve channel with malformed UUID": {
			owner:   email,
			thID:    wrongValue,
			channel: things.Channel{},
			err:     errors.ErrNotFound,
		},
	}

	for desc, tc := range cases {
		ch, err := chanRepo.RetrieveByThing(context.Background(), tc.owner, tc.thID)
		assert.Equal(t, tc.channel, ch, fmt.Sprintf("%s: expected %v got %v\n", desc, tc.channel, ch))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected no error got %d\n", desc, err))
	}
}

func TestChannelRemoval(t *testing.T) {
	email := "channel-removal@example.com"
	dbMiddleware := postgres.NewDatabase(db)
	chanRepo := postgres.NewChannelRepository(dbMiddleware)

	chID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	chs, err := chanRepo.Save(context.Background(), things.Channel{
		ID:    chID,
		Owner: email,
	})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	chID = chs[0].ID

	cases := map[string]struct {
		owner string
		chID  string
		err   error
	}{
		"remove non-existing channel": {
			owner: email,
			chID:  wrongValue,
			err:   errors.ErrRemoveEntity,
		},
		"remove channel": {
			owner: email,
			chID:  chID,
			err:   nil,
		},
	}

	for desc, tc := range cases {
		err := chanRepo.Remove(context.Background(), tc.owner, tc.chID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestConnect(t *testing.T) {
	email := "channel-connect@example.com"
	dbMiddleware := postgres.NewDatabase(db)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	thID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thID1, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thkey, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thkey1, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	th := []things.Thing{
		{
			ID:       thID,
			Owner:    email,
			Key:      thkey,
			Metadata: things.Metadata{},
		},
		{
			ID:       thID1,
			Owner:    email,
			Key:      thkey1,
			Metadata: things.Metadata{},
		}}

	ths, err := thingRepo.Save(context.Background(), th...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	thID = ths[0].ID

	chanRepo := postgres.NewChannelRepository(dbMiddleware)

	chID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	chs, err := chanRepo.Save(context.Background(), things.Channel{
		ID:    chID,
		Owner: email,
	})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	chID = chs[0].ID

	nonexistentThingID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	nonexistentChanID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc  string
		owner string
		chID  string
		thID  string
		err   error
	}{
		{
			desc:  "connect existing channel and thing",
			owner: email,
			chID:  chID,
			thID:  thID,
			err:   nil,
		},
		{
			desc:  "connect connected channel and thing",
			owner: email,
			chID:  chID,
			thID:  thID,
			err:   errors.ErrConflict,
		},
		{
			desc:  "connect with non-existing user",
			owner: wrongValue,
			chID:  chID,
			thID:  thID1,
			err:   errors.ErrNotFound,
		},
		{
			desc:  "connect non-existing channel",
			owner: email,
			chID:  nonexistentChanID,
			thID:  thID1,
			err:   errors.ErrNotFound,
		},
		{
			desc:  "connect non-existing thing",
			owner: email,
			chID:  chID,
			thID:  nonexistentThingID,
			err:   errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := chanRepo.Connect(context.Background(), tc.owner, tc.chID, []string{tc.thID})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestDisconnect(t *testing.T) {
	email := "channel-disconnect@example.com"
	dbMiddleware := postgres.NewDatabase(db)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	thID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thkey, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	th := things.Thing{
		ID:       thID,
		Owner:    email,
		Key:      thkey,
		Metadata: map[string]interface{}{},
	}
	ths, err := thingRepo.Save(context.Background(), th)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	thID = ths[0].ID

	chanRepo := postgres.NewChannelRepository(dbMiddleware)
	chID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	chs, err := chanRepo.Save(context.Background(), things.Channel{
		ID:    chID,
		Owner: email,
	})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	ch := chs[0]
	err = chanRepo.Connect(context.Background(), email, ch.ID, []string{thID})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	nonexistentThingID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	nonexistentChanID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc  string
		owner string
		chID  string
		thID  string
		err   error
	}{
		{
			desc:  "disconnect connected thing",
			owner: email,
			chID:  chID,
			thID:  thID,
			err:   nil,
		},
		{
			desc:  "disconnect non-connected thing",
			owner: email,
			chID:  chID,
			thID:  thID,
			err:   errors.ErrNotFound,
		},
		{
			desc:  "disconnect non-existing user",
			owner: wrongValue,
			chID:  chID,
			thID:  thID,
			err:   errors.ErrNotFound,
		},
		{
			desc:  "disconnect non-existing channel",
			owner: email,
			chID:  nonexistentChanID,
			thID:  thID,
			err:   errors.ErrNotFound,
		},
		{
			desc:  "disconnect non-existing thing",
			owner: email,
			chID:  chID,
			thID:  nonexistentThingID,
			err:   errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := chanRepo.Disconnect(context.Background(), tc.owner, tc.chID, []string{tc.thID})
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveConnByThingKey(t *testing.T) {
	email := "channel-access-check@example.com"
	dbMiddleware := postgres.NewDatabase(db)
	thingRepo := postgres.NewThingRepository(dbMiddleware)

	thID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	thkey, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	th := things.Thing{
		ID:    thID,
		Owner: email,
		Key:   thkey,
	}
	ths, err := thingRepo.Save(context.Background(), th)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	thID = ths[0].ID

	chanRepo := postgres.NewChannelRepository(dbMiddleware)
	chID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
	chs, err := chanRepo.Save(context.Background(), things.Channel{
		ID:    chID,
		Owner: email,
	})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	chID = chs[0].ID
	err = chanRepo.Connect(context.Background(), email, chID, []string{thID})
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := map[string]struct {
		chID      string
		key       string
		hasAccess bool
	}{
		"access check for thing that has access": {
			key:       th.Key,
			hasAccess: true,
		},
		"access check for thing without access": {
			key:       wrongValue,
			hasAccess: false,
		},
	}

	for desc, tc := range cases {
		_, err := chanRepo.RetrieveConnByThingKey(context.Background(), tc.key)
		hasAccess := err == nil
		assert.Equal(t, tc.hasAccess, hasAccess, fmt.Sprintf("%s: expected %t got %t\n", desc, tc.hasAccess, hasAccess))
	}
}

func TestRetrieveAll(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	chanRepo := postgres.NewChannelRepository(dbMiddleware)

	err := cleanTestTable(context.Background(), "channels", dbMiddleware)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	email := "channel-multi-retrieval@example.com"
	name := "channel_name"
	metadata := things.Metadata{
		"field": "value",
	}
	nameNum := uint64(3)
	metaNum := uint64(3)
	nameMetaNum := uint64(2)

	n := uint64(101)
	for i := uint64(0); i < n; i++ {
		chID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		ch := things.Channel{
			ID:    chID,
			Owner: email,
		}

		// Create Channels with name.
		if i < nameNum {
			ch.Name = fmt.Sprintf("%s-%d", name, i)
		}
		// Create Channels with metadata.
		if i >= nameNum && i < nameNum+metaNum {
			ch.Metadata = metadata
		}
		// Create Channels with name and metadata.
		if i >= n-nameMetaNum {
			ch.Metadata = metadata
			ch.Name = name
		}

		chanRepo.Save(context.Background(), ch)
	}

	cases := map[string]struct {
		size uint64
		err  error
	}{
		"retrieve all channels without limit": {
			size: n,
			err:  nil,
		},
	}

	for desc, tc := range cases {
		channels, err := chanRepo.RetrieveAll(context.Background())
		size := uint64(len(channels))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", desc, tc.size, size))
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %d\n", desc, err))
	}
}

func TestRetrieveAllConnections(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	chanRepo := postgres.NewChannelRepository(dbMiddleware)
	thingRepo := postgres.NewThingRepository(dbMiddleware)
	email := "Channel-connect@example.com"

	err := cleanTestTable(context.Background(), "connections", dbMiddleware)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	n := uint64(101)
	for i := uint64(0); i < n; i++ {
		thID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
		thkey, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		th := things.Thing{
			ID:       thID,
			Owner:    email,
			Key:      thkey,
			Metadata: things.Metadata{},
		}
		ths, err := thingRepo.Save(context.Background(), th)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
		thID = ths[0].ID

		chID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))
		chs, err := chanRepo.Save(context.Background(), things.Channel{
			ID:    chID,
			Owner: email,
		})
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
		chID = chs[0].ID

		err = chanRepo.Connect(context.Background(), email, chID, []string{thID})
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	}

	cases := map[string]struct {
		size uint64
		err  error
	}{
		"retrieve all channels": {
			size: n,
			err:  nil,
		},
	}

	for desc, tc := range cases {
		connections, err := chanRepo.RetrieveAllConnections(context.Background())
		size := uint64(len(connections))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d\n", desc, tc.size, size))
		assert.Nil(t, err, fmt.Sprintf("%s: expected no error got %d\n", desc, err))
	}
}

func testSortChannels(t *testing.T, pm things.PageMetadata, chs []things.Channel) {
	switch pm.Order {
	case "name":
		current := chs[0]
		for _, res := range chs {
			if pm.Dir == "asc" {
				assert.GreaterOrEqual(t, res.Name, current.Name)
			}
			if pm.Dir == "desc" {
				assert.GreaterOrEqual(t, current.Name, res.Name)
			}
			current = res
		}
	default:
		break
	}
}
