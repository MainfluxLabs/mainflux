// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis

import (
	"context"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/events"
	"github.com/go-redis/redis/v8"
)

const streamLen = 1000

type eventStore struct {
	auth.Service
	client *redis.Client
}

// NewEventStoreMiddleware returns wrapper around auth service that sends
// events to event store.
func NewEventStoreMiddleware(svc auth.Service, client *redis.Client) auth.Service {
	return eventStore{
		Service: svc,
		client:  client,
	}
}

func (es eventStore) CreateOrg(ctx context.Context, token string, org auth.Org) (auth.Org, error) {
	sorg, err := es.Service.CreateOrg(ctx, token, org)
	if err != nil {
		return sorg, err
	}

	event := createOrgEvent{
		id: sorg.ID,
	}
	record := &redis.XAddArgs{
		Stream:       events.AuthStream,
		MaxLenApprox: streamLen,
		Values:       event.Encode(),
	}
	es.client.XAdd(ctx, record).Err()

	return sorg, nil
}

func (es eventStore) RemoveOrgs(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		if err := es.Service.RemoveOrgs(ctx, token, id); err != nil {
			return err
		}

		event := removeOrgEvent{
			id: id,
		}
		record := &redis.XAddArgs{
			Stream:       events.AuthStream,
			MaxLenApprox: streamLen,
			Values:       event.Encode(),
		}
		es.client.XAdd(ctx, record).Err()
	}

	return nil
}
