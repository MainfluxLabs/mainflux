// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/things"
)

// Connection represents connection between channel and thing that is used for
// testing purposes.
type Connection struct {
	chanID    string
	thing     things.Thing
	connected bool
}

var _ things.ChannelRepository = (*channelRepositoryMock)(nil)

type channelRepositoryMock struct {
	mu       sync.Mutex
	counter  uint64
	channels map[string]things.Channel
	tconns   chan Connection                      // used for synchronization with thing repo
	cconns   map[string]map[string]things.Channel // used to track connections
	conns    map[string]string                    // used to track connections
	things   things.ThingRepository
}

// NewChannelRepository creates in-memory channel repository.
func NewChannelRepository(repo things.ThingRepository, tconns chan Connection) things.ChannelRepository {
	return &channelRepositoryMock{
		channels: make(map[string]things.Channel),
		tconns:   tconns,
		cconns:   make(map[string]map[string]things.Channel),
		things:   repo,
	}
}

func (crm *channelRepositoryMock) Save(_ context.Context, channels ...things.Channel) ([]things.Channel, error) {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	for i := range channels {
		crm.counter++
		if channels[i].ID == "" {
			channels[i].ID = fmt.Sprintf("%03d", crm.counter)
		}
		crm.channels[key(channels[i].OwnerID, channels[i].ID)] = channels[i]
	}

	return channels, nil
}

func (crm *channelRepositoryMock) Update(_ context.Context, channel things.Channel) error {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	dbKey := key(channel.OwnerID, channel.ID)

	if _, ok := crm.channels[dbKey]; !ok {
		return errors.ErrNotFound
	}

	crm.channels[dbKey] = channel
	return nil
}

func (crm *channelRepositoryMock) RetrieveByID(_ context.Context, id string) (things.Channel, error) {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	for _, ch := range crm.channels {
		if ch.ID == id {
			return ch, nil
		}
	}

	return things.Channel{}, errors.ErrNotFound
}

func (crm *channelRepositoryMock) RetrieveByOwner(_ context.Context, owner string, pm things.PageMetadata) (things.ChannelsPage, error) {
	if pm.Limit < 0 {
		return things.ChannelsPage{}, nil
	}

	first := int(pm.Offset)
	last := first + int(pm.Limit)

	var chs []things.Channel

	// This obscure way to examine map keys is enforced by the key structure
	// itself (see mocks/commons.go).
	prefix := fmt.Sprintf("%s-", owner)
	for k, v := range crm.channels {
		if strings.HasPrefix(k, prefix) {
			chs = append(chs, v)
		}
	}

	// Sort Channels list
	chs = sortChannels(pm, chs)

	if last > len(chs) || last == 0 {
		last = len(chs)
	}

	if first > last {
		return things.ChannelsPage{}, nil
	}

	page := things.ChannelsPage{
		Channels: chs[first:last],
		PageMetadata: things.PageMetadata{
			Total:  crm.counter,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (crm *channelRepositoryMock) RetrieveByAdmin(ctx context.Context, pm things.PageMetadata) (things.ChannelsPage, error) {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	if pm.Limit < 0 {
		return things.ChannelsPage{}, nil
	}

	i := uint64(0)
	var chs []things.Channel
	for _, ch := range crm.channels {
		if i >= pm.Offset && i < pm.Offset+pm.Limit {
			chs = append(chs, ch)
		}
		i++
	}

	page := things.ChannelsPage{
		Channels: chs,
		PageMetadata: things.PageMetadata{
			Total:  crm.counter,
			Offset: pm.Offset,
			Limit:  pm.Limit,
		},
	}

	return page, nil
}

func (crm *channelRepositoryMock) RetrieveByThing(_ context.Context, thID string) (things.Channel, error) {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	for _, ch := range crm.channels {
		for _, co := range crm.cconns[thID] {
			if ch.ID == co.ID {
				return ch, nil
			}
		}
	}

	return things.Channel{}, errors.ErrNotFound
}

func (crm *channelRepositoryMock) RetrieveConns(_ context.Context, thID string, pm things.PageMetadata) (things.ChannelsPage, error) {
	return things.ChannelsPage{}, nil
}

func (crm *channelRepositoryMock) Remove(_ context.Context, owner string, ids ...string) error {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	for _, id := range ids {
		if _, ok := crm.channels[key(owner, id)]; !ok {
			return errors.ErrNotFound
		}

		delete(crm.channels, key(owner, id))

		for thk := range crm.cconns {
			delete(crm.cconns[thk], key(owner, id))
		}
		crm.tconns <- Connection{
			chanID:    id,
			connected: false,
		}
	}

	return nil
}

func (crm *channelRepositoryMock) Connect(_ context.Context, chID string, thIDs []string) error {
	ch, err := crm.RetrieveByID(context.Background(), chID)
	if err != nil {
		return err
	}

	for _, thID := range thIDs {
		if _, ok := crm.cconns[thID]; ok {
			return errors.ErrConflict
		}
		th, err := crm.things.RetrieveByID(context.Background(), thID)
		if err != nil {
			return err
		}
		crm.tconns <- Connection{
			chanID:    chID,
			thing:     th,
			connected: true,
		}
		if _, ok := crm.cconns[thID]; !ok {
			crm.cconns[thID] = make(map[string]things.Channel)
		}
		crm.cconns[thID][chID] = ch
	}

	return nil
}

func (crm *channelRepositoryMock) Disconnect(_ context.Context, chID string, thIDs []string) error {
	for _, thID := range thIDs {
		if _, ok := crm.cconns[thID]; !ok {
			return errors.ErrNotFound
		}

		if _, ok := crm.cconns[thID][chID]; !ok {
			return errors.ErrNotFound
		}

		crm.tconns <- Connection{
			chanID:    chID,
			thing:     things.Thing{ID: thID},
			connected: false,
		}
		delete(crm.cconns[thID], chID)
	}

	return nil
}

func (crm *channelRepositoryMock) RetrieveConnByThingKey(_ context.Context, token string) (things.Connection, error) {
	tid, err := crm.things.RetrieveByKey(context.Background(), token)
	if err != nil {
		return things.Connection{}, err
	}

	chans, ok := crm.cconns[tid]
	if !ok {
		return things.Connection{}, errors.ErrAuthorization
	}

	if len(chans) == 0 {
		return things.Connection{}, errors.ErrAuthorization
	}

	for _, v := range chans {
		return things.Connection{ThingID: tid, ChannelID: v.ID}, nil
	}

	return things.Connection{}, errors.ErrNotFound
}

func (crm *channelRepositoryMock) HasThingByID(_ context.Context, chanID, thingID string) error {
	chans, ok := crm.cconns[thingID]
	if !ok {
		return errors.ErrAuthorization
	}

	if _, ok := chans[chanID]; !ok {
		return errors.ErrAuthorization
	}

	return nil
}

func (crm *channelRepositoryMock) RetrieveAll(ctx context.Context) ([]things.Channel, error) {
	crm.mu.Lock()
	defer crm.mu.Unlock()

	var chs []things.Channel
	for _, v := range crm.channels {
		chs = append(chs, v)
	}

	return chs, nil
}

func (crm *channelRepositoryMock) RetrieveAllConnections(ctx context.Context) ([]things.Connection, error) {
	crm.mu.Lock()
	defer crm.mu.Unlock()
	var conns []things.Connection

	for thingID, con := range crm.cconns {
		for _, v := range con {
			con := things.Connection{
				ChannelID: v.ID,
				ThingID:   thingID,
			}
			conns = append(conns, con)
		}
	}

	return conns, nil

}

type channelCacheMock struct {
	mu       sync.Mutex
	channels map[string]string
}

// NewChannelCache returns mock cache instance.
func NewChannelCache() things.ChannelCache {
	return &channelCacheMock{
		channels: make(map[string]string),
	}
}

func (ccm *channelCacheMock) Connect(_ context.Context, chanID, thingID string) error {
	ccm.mu.Lock()
	defer ccm.mu.Unlock()

	ccm.channels[chanID] = thingID
	return nil
}

func (ccm *channelCacheMock) HasThing(_ context.Context, chanID, thingID string) bool {
	ccm.mu.Lock()
	defer ccm.mu.Unlock()

	return ccm.channels[chanID] == thingID
}

func (ccm *channelCacheMock) Disconnect(_ context.Context, chanID, thingID string) error {
	ccm.mu.Lock()
	defer ccm.mu.Unlock()

	delete(ccm.channels, chanID)
	return nil
}

func (ccm *channelCacheMock) Remove(_ context.Context, chanID string) error {
	ccm.mu.Lock()
	defer ccm.mu.Unlock()

	delete(ccm.channels, chanID)
	return nil
}
