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
	saveThings                 = "save_things"
	updateThing                = "update_thing"
	updateThingGroupAndProfile = "update_thing_group_and_profile"
	retrieveThingByID          = "retrieve_thing_by_id"
	retrieveThingByKey         = "retrieve_thing_by_key"
	retrieveThingsByProfile    = "retrieve_things_by_profile"
	retrieveThingsByGroups     = "retrieve_things_by_groups"
	removeThing                = "remove_thing"
	retrieveAllThings          = "retrieve_all_things"
	backupAllThings            = "backup_all_things"
	updateExternalKey          = "update_external_key"
	removeExternalKey          = "remove_external_key"
)

var _ things.ThingRepository = (*thingRepositoryMiddleware)(nil)

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
