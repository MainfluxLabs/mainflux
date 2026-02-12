// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/opentracing/opentracing-go"
)

const (
	saveThingKey             = "save_thing_key"
	retrieveThingIDByKey     = "retrieve_thing_id_by_key"
	removeThingKey           = "remove_thing_key"
	saveThingGroup           = "save_thing_group"
	retrieveGroupIDByThingID = "retrieve_group_id_by_thing_id"
	removeThingGroup         = "remove_thing_group"
)

var _ things.ThingCache = (*thingCacheMiddleware)(nil)

type thingCacheMiddleware struct {
	tracer opentracing.Tracer
	cache  things.ThingCache
}

// ThingCacheMiddleware tracks request and their latency, and adds spans
// to context.
func ThingCacheMiddleware(tracer opentracing.Tracer, cache things.ThingCache) things.ThingCache {
	return thingCacheMiddleware{
		tracer: tracer,
		cache:  cache,
	}
}

func (tcm thingCacheMiddleware) Save(ctx context.Context, key things.ThingKey, thingID string) error {
	span := dbutil.CreateSpan(ctx, tcm.tracer, saveThingKey)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return tcm.cache.Save(ctx, key, thingID)
}

func (tcm thingCacheMiddleware) ID(ctx context.Context, key things.ThingKey) (string, error) {
	span := dbutil.CreateSpan(ctx, tcm.tracer, retrieveThingIDByKey)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return tcm.cache.ID(ctx, key)
}

func (tcm thingCacheMiddleware) RemoveThing(ctx context.Context, thingID string) error {
	span := dbutil.CreateSpan(ctx, tcm.tracer, removeThing)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return tcm.cache.RemoveThing(ctx, thingID)
}

func (tcm thingCacheMiddleware) RemoveKey(ctx context.Context, key things.ThingKey) error {
	span := dbutil.CreateSpan(ctx, tcm.tracer, removeThingKey)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return tcm.cache.RemoveKey(ctx, key)
}

func (tcm thingCacheMiddleware) SaveGroup(ctx context.Context, thingID string, groupID string) error {
	span := dbutil.CreateSpan(ctx, tcm.tracer, saveThingGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return tcm.cache.SaveGroup(ctx, thingID, groupID)
}

func (tcm thingCacheMiddleware) ViewGroup(ctx context.Context, thingID string) (string, error) {
	span := dbutil.CreateSpan(ctx, tcm.tracer, retrieveGroupIDByThingID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return tcm.cache.ViewGroup(ctx, thingID)
}

func (tcm thingCacheMiddleware) RemoveGroup(ctx context.Context, thingID string) error {
	span := dbutil.CreateSpan(ctx, tcm.tracer, removeThingGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return tcm.cache.RemoveGroup(ctx, thingID)
}
