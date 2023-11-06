package postgres_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/mqtt"
	"github.com/MainfluxLabs/mainflux/mqtt/postgres"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	numSubs   = 100
	subtopic  = "subtopic"
	invalidID = "invalid"
	noLimit   = 0
)

func TestSave(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.NewRepository(dbMiddleware)

	chanID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	thingID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	sub := mqtt.Subscription{
		Subtopic: subtopic,
		ThingID:  thingID,
		ChanID:   chanID,
	}

	sub2 := sub
	sub2.Subtopic = "subtopic_2"

	invalidSub := sub
	invalidSub.ThingID = invalidID

	cases := []struct {
		desc string
		sub  mqtt.Subscription
		err  error
	}{

		{
			desc: "save subscription successfully",
			sub:  sub,
			err:  nil,
		},
		{
			desc: "subscribe thing to several subtopics successfully",
			sub:  sub2,
			err:  nil,
		},
		{
			desc: "save existing subscription",
			sub:  sub,
			err:  errors.ErrConflict,
		},
		{
			desc: "save invalid subscription",
			sub:  invalidSub,
			err:  errors.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		err := repo.Save(context.Background(), tc.sub)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRemove(t *testing.T) {
	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.NewRepository(dbMiddleware)

	chanID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	thingID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	invalidID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	sub := mqtt.Subscription{
		Subtopic: subtopic,
		ThingID:  thingID,
		ChanID:   chanID,
		ClientID: "client-id-1",
	}

	nonExistingSub := sub
	nonExistingSub.ThingID = invalidID

	err = repo.Save(context.Background(), sub)
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	cases := []struct {
		desc string
		sub  mqtt.Subscription
		err  error
	}{
		{
			desc: "remove successfully",
			sub:  sub,
			err:  nil,
		},
		{
			desc: "remove non-existing subscription",
			sub:  nonExistingSub,
			err:  nil,
		},
	}

	for _, tc := range cases {
		err := repo.Remove(context.Background(), tc.sub)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestRetrieveByID(t *testing.T) {
	_, err := db.Exec("DELETE FROM subscriptions")
	require.Nil(t, err, fmt.Sprintf("cleanup must not fail: %s", err))

	dbMiddleware := postgres.NewDatabase(db)
	repo := postgres.NewRepository(dbMiddleware)

	var subs []mqtt.Subscription

	chanID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	nonExistingChanID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	for i := 0; i < numSubs; i++ {
		thID, err := idProvider.ID()
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		sub := mqtt.Subscription{
			Subtopic: subtopic,
			ThingID:  thID,
			ChanID:   chanID,
			ClientID: fmt.Sprintf("client-id-%d", i),
		}

		err = repo.Save(context.Background(), sub)
		require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

		subs = append(subs, sub)
	}

	cases := []struct {
		desc      string
		size      int
		channelID string
		pageMeta  mqtt.PageMetadata
		page      mqtt.Page
		err       error
	}{
		{
			desc:      "retrieve all subscriptions for existing channel",
			size:      10,
			channelID: chanID,
			pageMeta: mqtt.PageMetadata{
				Total:  numSubs,
				Offset: 0,
				Limit:  10,
			},
			page: mqtt.Page{
				PageMetadata: mqtt.PageMetadata{
					Total:  numSubs,
					Offset: 0,
					Limit:  10,
				},
				Subscriptions: subs[0:10],
			},
			err: nil,
		},
		{
			desc:      "retrieve all subscriptions for existing channel with no limit",
			size:      numSubs,
			channelID: chanID,
			pageMeta: mqtt.PageMetadata{
				Total: numSubs,
				Limit: 0,
			},
			page: mqtt.Page{
				PageMetadata: mqtt.PageMetadata{
					Total: numSubs,
					Limit: noLimit,
				},
				Subscriptions: subs,
			},
			err: nil,
		},
		{
			desc:      "retrieve subscriptions with non-existing channel",
			size:      0,
			channelID: nonExistingChanID,
			pageMeta: mqtt.PageMetadata{
				Total:  0,
				Offset: 0,
				Limit:  noLimit,
			},
			page: mqtt.Page{
				PageMetadata: mqtt.PageMetadata{
					Total:  0,
					Offset: 0,
					Limit:  noLimit,
				},
				Subscriptions: nil,
			},
			err: nil,
		},
		{
			desc:      "retrieve subscriptions with invalid channel",
			size:      0,
			channelID: invalidID,
			pageMeta:  mqtt.PageMetadata{},
			page: mqtt.Page{
				PageMetadata: mqtt.PageMetadata{
					Total:  0,
					Offset: 0,
					Limit:  noLimit,
				},
				Subscriptions: nil,
			},
			err: errors.ErrRetrieveEntity,
		},
	}

	for _, tc := range cases {
		page, err := repo.RetrieveByChannelID(context.Background(), tc.pageMeta, tc.channelID)
		size := len(page.Subscriptions)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
		assert.Equal(t, tc.pageMeta.Total, page.Total, fmt.Sprintf("%s: expected total %d got %d\n", tc.desc, tc.pageMeta.Total, page.Total))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected %v got %v\n", tc.desc, tc.size, size))
	}
}
