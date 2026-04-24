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

// DefaultStreamMaxLen is the default approximate retention (number of
// entries) for the auth event stream. Overridable per-deployment via the
// MaxLen argument to NewEventStoreMiddleware.
const DefaultStreamMaxLen = 100000

type eventStore struct {
	auth.Service
	client *redis.Client
	logger logger.Logger
	maxLen int64
}

// NewEventStoreMiddleware returns wrapper around auth service that sends
// events to event store. maxLen controls the approximate stream retention;
// pass 0 to use DefaultStreamMaxLen.
func NewEventStoreMiddleware(svc auth.Service, client *redis.Client, maxLen int64, logger logger.Logger) auth.Service {
	if maxLen <= 0 {
		maxLen = DefaultStreamMaxLen
	}
	return eventStore{
		Service: svc,
		client:  client,
		logger:  logger,
		maxLen:  maxLen,
	}
}

func (es eventStore) publish(ctx context.Context, e events.Event) {
	vals := e.Encode()
	record := &redis.XAddArgs{
		Stream:       events.AuthStream,
		MaxLenApprox: es.maxLen,
		Values:       map[string]any(vals),
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
