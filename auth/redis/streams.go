// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis

import (
	"context"
	"fmt"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/events"
	"github.com/go-redis/redis/v8"
)

const streamLen = 1000

type eventStore struct {
	auth.Service
	client *redis.Client
	logger logger.Logger
}

// NewEventStoreMiddleware returns wrapper around auth service that sends
// events to event store.
func NewEventStoreMiddleware(svc auth.Service, client *redis.Client, logger logger.Logger) auth.Service {
	return eventStore{
		Service: svc,
		client:  client,
		logger:  logger,
	}
}

func (es eventStore) publish(ctx context.Context, e events.Event) {
	vals := e.Encode()
	record := &redis.XAddArgs{
		Stream:       events.AuthStream,
		MaxLenApprox: streamLen,
		Values:       vals,
	}
	if err := es.client.XAdd(ctx, record).Err(); err != nil {
		es.logger.Warn(fmt.Sprintf("failed to publish %s event: %s", vals.Operation(), err))
	}
}

func (es eventStore) CreateOrg(ctx context.Context, token string, org auth.Org) (auth.Org, error) {
	sorg, err := es.Service.CreateOrg(ctx, token, org)
	if err != nil {
		return sorg, err
	}

	es.publish(ctx, events.OrgCreated{ID: sorg.ID})

	return sorg, nil
}

func (es eventStore) RemoveOrgs(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		if err := es.Service.RemoveOrgs(ctx, token, id); err != nil {
			return err
		}

		es.publish(ctx, events.OrgRemoved{ID: id})
	}

	return nil
}
