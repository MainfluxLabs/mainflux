// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/opentracing/opentracing-go"
)

const (
	saveThing                  = "save_thing"
	saveThings                 = "save_things"
	updateThing                = "update_thing"
	updateThingGroupAndProfile = "update_thing_group_and_profile"
	retrieveThingByID          = "retrieve_thing_by_id"
	retrieveThingByKey         = "retrieve_thing_by_key"
	retrieveThingsByProfile    = "retrieve_things_by_profile"
	retrieveThingsByGroups     = "retrieve_things_by_groups"
	removeThing                = "remove_thing"
	removeKey                  = "remove_key"
	retrieveThingIDByKey       = "retrieve_id_by_key"
	retrieveAllThings          = "retrieve_all_things"
	backupAllThings            = "backup_all_things"
	saveGroupIDByThingID       = "save_group_id_by_thing_id"
	retrieveGroupIDByThingID   = "retrieve_group_id_by_thing_id"
	removeGroupIDByThingID     = "remove_group_id_by_thing_id"
	updateExternalKey          = "update_external_key"
	removeExternalKey          = "remove_external_key"
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
	span := dbutil.CreateSpan(ctx, trm.tracer, saveThings)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.Save(ctx, ths...)
}

func (trm thingRepositoryMiddleware) Update(ctx context.Context, th things.Thing) error {
	span := dbutil.CreateSpan(ctx, trm.tracer, updateThing)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.Update(ctx, th)
}

func (trm thingRepositoryMiddleware) UpdateGroupAndProfile(ctx context.Context, th things.Thing) error {
	span := dbutil.CreateSpan(ctx, trm.tracer, updateThingGroupAndProfile)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.Update(ctx, th)
}

func (trm thingRepositoryMiddleware) RetrieveByID(ctx context.Context, id string) (things.Thing, error) {
	span := dbutil.CreateSpan(ctx, trm.tracer, retrieveThingByID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.RetrieveByID(ctx, id)
}

func (trm thingRepositoryMiddleware) RetrieveByKey(ctx context.Context, key things.ThingKey) (string, error) {
	span := dbutil.CreateSpan(ctx, trm.tracer, retrieveThingByKey)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.RetrieveByKey(ctx, key)
}

func (trm thingRepositoryMiddleware) RetrieveByGroups(ctx context.Context, ids []string, pm apiutil.PageMetadata) (things.ThingsPage, error) {
	span := dbutil.CreateSpan(ctx, trm.tracer, retrieveThingsByGroups)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.RetrieveByGroups(ctx, ids, pm)
}

func (trm thingRepositoryMiddleware) RetrieveByProfile(ctx context.Context, chID string, pm apiutil.PageMetadata) (things.ThingsPage, error) {
	span := dbutil.CreateSpan(ctx, trm.tracer, retrieveThingsByProfile)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.RetrieveByProfile(ctx, chID, pm)
}

func (trm thingRepositoryMiddleware) Remove(ctx context.Context, ids ...string) error {
	span := dbutil.CreateSpan(ctx, trm.tracer, removeThing)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.Remove(ctx, ids...)
}

func (trm thingRepositoryMiddleware) BackupAll(ctx context.Context) ([]things.Thing, error) {
	span := dbutil.CreateSpan(ctx, trm.tracer, backupAllThings)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.BackupAll(ctx)
}

func (trm thingRepositoryMiddleware) RetrieveAll(ctx context.Context, pm apiutil.PageMetadata) (things.ThingsPage, error) {
	span := dbutil.CreateSpan(ctx, trm.tracer, retrieveAllThings)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.RetrieveAll(ctx, pm)
}

func (trm thingRepositoryMiddleware) UpdateExternalKey(ctx context.Context, key, thingID string) error {
	span := dbutil.CreateSpan(ctx, trm.tracer, updateExternalKey)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.UpdateExternalKey(ctx, key, thingID)
}

func (trm thingRepositoryMiddleware) RemoveExternalKey(ctx context.Context, thingID string) error {
	span := dbutil.CreateSpan(ctx, trm.tracer, removeExternalKey)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return trm.repo.RemoveExternalKey(ctx, thingID)
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

func (tcm thingCacheMiddleware) Save(ctx context.Context, key things.ThingKey, thingID string) error {
	span := dbutil.CreateSpan(ctx, tcm.tracer, saveThing)
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
	span := dbutil.CreateSpan(ctx, tcm.tracer, removeKey)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return tcm.cache.RemoveKey(ctx, key)
}

func (tcm thingCacheMiddleware) SaveGroup(ctx context.Context, thingID string, groupID string) error {
	span := dbutil.CreateSpan(ctx, tcm.tracer, saveGroupIDByThingID)
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
	span := dbutil.CreateSpan(ctx, tcm.tracer, removeGroupIDByThingID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return tcm.cache.RemoveGroup(ctx, thingID)
}
