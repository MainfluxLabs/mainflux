// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/things"
	opentracing "github.com/opentracing/opentracing-go"
)

const (
	saveThingOp                = "save_thing"
	saveThingsOp               = "save_things"
	updateThingOp              = "update_thing"
	updateThingKeyOp           = "update_thing_by_key"
	retrieveThingByIDOp        = "retrieve_thing_by_id"
	retrieveThingByKeyOp       = "retrieve_thing_by_key"
	retrieveThingsByProfileOp  = "retrieve_things_by_profile"
	retrieveThingsByGroupIDsOp = "retrieve_things_by_group_ids"
	removeThingOp              = "remove_thing"
	retrieveThingIDByKeyOp     = "retrieve_id_by_key"
	retrieveAllThingsOp        = "retrieve_all_things"
	saveGroupIDByThingIDOp     = "save_group_id_by_thing_id"
	retrieveGroupIDByThingIDOp = "retrieve_group_id_by_thing_id"
	removeGroupIDByThingIDOp   = "remove_group_id_by_thing_id"
)

var (
	_ things.ThingRepository = (*thingRepositoryMiddleware)(nil)
	_ things.ThingCache      = (*thingCacheMiddleware)(nil)
)

type thingRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   things.ThingRepository
}

// ThingRepositoryMiddleware tracks request and their latency, and adds spans
// to context.
func ThingRepositoryMiddleware(tracer opentracing.Tracer, repo things.ThingRepository) things.ThingRepository {
	return thingRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (trm thingRepositoryMiddleware) Save(ctx context.Context, ths ...things.Thing) ([]things.Thing, error) {
	span := createSpan(ctx, trm.tracer, saveThingsOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.Save(ctx, ths...)
}

func (trm thingRepositoryMiddleware) Update(ctx context.Context, th things.Thing) error {
	span := createSpan(ctx, trm.tracer, updateThingOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.Update(ctx, th)
}

func (trm thingRepositoryMiddleware) UpdateKey(ctx context.Context, id, key string) error {
	span := createSpan(ctx, trm.tracer, updateThingKeyOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.UpdateKey(ctx, id, key)
}

func (trm thingRepositoryMiddleware) RetrieveByID(ctx context.Context, id string) (things.Thing, error) {
	span := createSpan(ctx, trm.tracer, retrieveThingByIDOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.RetrieveByID(ctx, id)
}

func (trm thingRepositoryMiddleware) RetrieveByKey(ctx context.Context, key string) (string, error) {
	span := createSpan(ctx, trm.tracer, retrieveThingByKeyOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.RetrieveByKey(ctx, key)
}

func (trm thingRepositoryMiddleware) RetrieveByGroupIDs(ctx context.Context, ids []string, pm things.PageMetadata) (things.ThingsPage, error) {
	span := createSpan(ctx, trm.tracer, retrieveThingsByGroupIDsOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.RetrieveByGroupIDs(ctx, ids, pm)
}

func (trm thingRepositoryMiddleware) RetrieveByProfile(ctx context.Context, chID string, pm things.PageMetadata) (things.ThingsPage, error) {
	span := createSpan(ctx, trm.tracer, retrieveThingsByProfileOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.RetrieveByProfile(ctx, chID, pm)
}

func (trm thingRepositoryMiddleware) Remove(ctx context.Context, ids ...string) error {
	span := createSpan(ctx, trm.tracer, removeThingOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.Remove(ctx, ids...)
}

func (trm thingRepositoryMiddleware) RetrieveAll(ctx context.Context) ([]things.Thing, error) {
	span := createSpan(ctx, trm.tracer, retrieveAllThingsOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.RetrieveAll(ctx)
}

func (trm thingRepositoryMiddleware) RetrieveByAdmin(ctx context.Context, pm things.PageMetadata) (things.ThingsPage, error) {
	span := createSpan(ctx, trm.tracer, retrieveAllThingsOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.RetrieveByAdmin(ctx, pm)
}

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

func (tcm thingCacheMiddleware) Save(ctx context.Context, thingKey string, thingID string) error {
	span := createSpan(ctx, tcm.tracer, saveThingOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return tcm.cache.Save(ctx, thingKey, thingID)
}

func (tcm thingCacheMiddleware) ID(ctx context.Context, thingKey string) (string, error) {
	span := createSpan(ctx, tcm.tracer, retrieveThingIDByKeyOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return tcm.cache.ID(ctx, thingKey)
}

func (tcm thingCacheMiddleware) Remove(ctx context.Context, thingID string) error {
	span := createSpan(ctx, tcm.tracer, removeThingOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return tcm.cache.Remove(ctx, thingID)
}

func (tcm thingCacheMiddleware) SaveGroup(ctx context.Context, thingID string, groupID string) error {
	span := createSpan(ctx, tcm.tracer, saveGroupIDByThingIDOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return tcm.cache.SaveGroup(ctx, thingID, groupID)
}

func (tcm thingCacheMiddleware) ViewGroup(ctx context.Context, thingID string) (string, error) {
	span := createSpan(ctx, tcm.tracer, retrieveGroupIDByThingIDOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return tcm.cache.ViewGroup(ctx, thingID)
}

func (tcm thingCacheMiddleware) RemoveGroup(ctx context.Context, thingID string) error {
	span := createSpan(ctx, tcm.tracer, removeGroupIDByThingIDOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return tcm.cache.RemoveGroup(ctx, thingID)
}

func createSpan(ctx context.Context, tracer opentracing.Tracer, opName string) opentracing.Span {
	if parentSpan := opentracing.SpanFromContext(ctx); parentSpan != nil {
		return tracer.StartSpan(
			opName,
			opentracing.ChildOf(parentSpan.Context()),
		)
	}
	return tracer.StartSpan(opName)
}
