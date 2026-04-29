// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package redis

import (
	"context"

	"github.com/MainfluxLabs/mainflux/auth"
	"github.com/MainfluxLabs/mainflux/pkg/events"
)

type eventStore struct {
	auth.Service
	pub events.Publisher
}

func NewEventStoreMiddleware(svc auth.Service, pub events.Publisher) auth.Service {
	return eventStore{
		Service: svc,
		pub:     pub,
	}
}

func (es eventStore) CreateOrg(ctx context.Context, token string, org auth.Org) (auth.Org, error) {
	sorg, err := es.Service.CreateOrg(ctx, token, org)
	if err != nil {
		return sorg, err
	}

	es.pub.Publish(ctx, events.OrgCreated{ID: sorg.ID})

	return sorg, nil
}

func (es eventStore) RemoveOrgs(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		if err := es.Service.RemoveOrgs(ctx, token, id); err != nil {
			return err
		}

		es.pub.Publish(ctx, events.OrgRemoved{ID: id})
	}

	return nil
}
