// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/things"
	opentracing "github.com/opentracing/opentracing-go"
)

const (
	saveChannelsOp           = "save_channels"
	updateChannelOp          = "update_channel"
	retrieveChannelByIDOp    = "retrieve_channel_by_id"
	retrieveByThingOp        = "retrieve_by_thing"
	retrieveChannelConnsOp   = "retrieve_channels_conns"
	removeChannelOp          = "retrieve_channel"
	connectOp                = "connect"
	disconnectOp             = "disconnect"
	hasThingOp               = "has_thing"
	hasThingByIDOp           = "has_thing_by_id"
	retrieveAllChannelsOp    = "retrieve_all_channels"
	retrieveAllConnectionsOp = "retrieve_all_connections"
)

var (
	_ things.ChannelRepository = (*channelRepositoryMiddleware)(nil)
	_ things.ChannelCache      = (*channelCacheMiddleware)(nil)
)

type channelRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   things.ChannelRepository
}

// ChannelRepositoryMiddleware tracks request and their latency, and adds spans
// to context.
func ChannelRepositoryMiddleware(tracer opentracing.Tracer, repo things.ChannelRepository) things.ChannelRepository {
	return channelRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (crm channelRepositoryMiddleware) Save(ctx context.Context, channels ...things.Channel) ([]things.Channel, error) {
	span := createSpan(ctx, crm.tracer, saveChannelsOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.Save(ctx, channels...)
}

func (crm channelRepositoryMiddleware) Update(ctx context.Context, ch things.Channel) error {
	span := createSpan(ctx, crm.tracer, updateChannelOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.Update(ctx, ch)
}

func (crm channelRepositoryMiddleware) RetrieveByID(ctx context.Context, id string) (things.Channel, error) {
	span := createSpan(ctx, crm.tracer, retrieveChannelByIDOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RetrieveByID(ctx, id)
}

func (crm channelRepositoryMiddleware) RetrieveByOwner(ctx context.Context, owner string, pm things.PageMetadata) (things.ChannelsPage, error) {
	span := createSpan(ctx, crm.tracer, retrieveByOwnerOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RetrieveByOwner(ctx, owner, pm)
}

func (crm channelRepositoryMiddleware) RetrieveByThing(ctx context.Context, thID string) (things.Channel, error) {
	span := createSpan(ctx, crm.tracer, retrieveByThingOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RetrieveByThing(ctx, thID)
}

func (crm channelRepositoryMiddleware) Remove(ctx context.Context, owner string, ids ...string) error {
	span := createSpan(ctx, crm.tracer, removeChannelOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.Remove(ctx, owner, ids...)
}

func (crm channelRepositoryMiddleware) Connect(ctx context.Context, chID string, thIDs []string) error {
	span := createSpan(ctx, crm.tracer, connectOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.Connect(ctx, chID, thIDs)
}

func (crm channelRepositoryMiddleware) Disconnect(ctx context.Context, chID string, thIDs []string) error {
	span := createSpan(ctx, crm.tracer, disconnectOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.Disconnect(ctx, chID, thIDs)
}

func (crm channelRepositoryMiddleware) RetrieveConnByThingKey(ctx context.Context, key string) (things.Connection, error) {
	span := createSpan(ctx, crm.tracer, hasThingOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RetrieveConnByThingKey(ctx, key)
}

func (crm channelRepositoryMiddleware) RetrieveAll(ctx context.Context) ([]things.Channel, error) {
	span := createSpan(ctx, crm.tracer, retrieveAllChannelsOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RetrieveAll(ctx)
}

func (crm channelRepositoryMiddleware) RetrieveByAdmin(ctx context.Context, pm things.PageMetadata) (things.ChannelsPage, error) {
	span := createSpan(ctx, crm.tracer, retrieveAllChannelsOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RetrieveByAdmin(ctx, pm)
}

func (crm channelRepositoryMiddleware) RetrieveAllConnections(ctx context.Context) ([]things.Connection, error) {
	span := createSpan(ctx, crm.tracer, retrieveAllConnectionsOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RetrieveAllConnections(ctx)
}

type channelCacheMiddleware struct {
	tracer opentracing.Tracer
	cache  things.ChannelCache
}

// ChannelCacheMiddleware tracks request and their latency, and adds spans
// to context.
func ChannelCacheMiddleware(tracer opentracing.Tracer, cache things.ChannelCache) things.ChannelCache {
	return channelCacheMiddleware{
		tracer: tracer,
		cache:  cache,
	}
}

func (ccm channelCacheMiddleware) Connect(ctx context.Context, chanID, thingID string) error {
	span := createSpan(ctx, ccm.tracer, connectOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return ccm.cache.Connect(ctx, chanID, thingID)
}

func (ccm channelCacheMiddleware) HasThing(ctx context.Context, chanID, thingID string) bool {
	span := createSpan(ctx, ccm.tracer, hasThingOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return ccm.cache.HasThing(ctx, chanID, thingID)
}

func (ccm channelCacheMiddleware) Disconnect(ctx context.Context, chanID, thingID string) error {
	span := createSpan(ctx, ccm.tracer, disconnectOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return ccm.cache.Disconnect(ctx, chanID, thingID)
}

func (ccm channelCacheMiddleware) Remove(ctx context.Context, chanID string) error {
	span := createSpan(ctx, ccm.tracer, removeChannelOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return ccm.cache.Remove(ctx, chanID)
}
