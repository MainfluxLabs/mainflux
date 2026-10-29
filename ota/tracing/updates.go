// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/ota"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/opentracing/opentracing-go"
)

const (
	saveUpdate                = "save_update"
	retrieveUpdateByID        = "retrieve_update_by_id"
	retrieveUpdatesByGroup    = "retrieve_updates_by_group"
	removeUpdate              = "remove_update"
	removeUpdatesByGroup      = "remove_updates_by_group"
	retrieveThingUpdates      = "retrieve_thing_updates"
	retrieveThingUpdate       = "retrieve_thing_update"
	retrieveActiveThingUpdate = "retrieve_active_thing_update"
	updateThingUpdate         = "update_thing_update"
	retryFailedThingUpdates   = "retry_failed_thing_updates"
	failTimedOutThingUpdates  = "fail_timed_out_thing_updates"
	removeThingUpdatesByThing = "remove_thing_updates_by_thing"
)

var _ ota.UpdateRepository = (*updateRepositoryMiddleware)(nil)

type updateRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   ota.UpdateRepository
}

// UpdateRepositoryMiddleware tracks request and their latency, and adds spans
// to context.
func UpdateRepositoryMiddleware(tracer opentracing.Tracer, repo ota.UpdateRepository) ota.UpdateRepository {
	return updateRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (urm updateRepositoryMiddleware) Save(ctx context.Context, u ota.Update, thingIDs ...string) error {
	span := dbutil.CreateSpan(ctx, urm.tracer, saveUpdate)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.Save(ctx, u, thingIDs...)
}

func (urm updateRepositoryMiddleware) RetrieveByID(ctx context.Context, id string) (ota.Update, error) {
	span := dbutil.CreateSpan(ctx, urm.tracer, retrieveUpdateByID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.RetrieveByID(ctx, id)
}

func (urm updateRepositoryMiddleware) RetrieveByGroup(ctx context.Context, groupID string, pm ota.PageMetadata) (ota.UpdatesPage, error) {
	span := dbutil.CreateSpan(ctx, urm.tracer, retrieveUpdatesByGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.RetrieveByGroup(ctx, groupID, pm)
}

func (urm updateRepositoryMiddleware) Remove(ctx context.Context, id string) error {
	span := dbutil.CreateSpan(ctx, urm.tracer, removeUpdate)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.Remove(ctx, id)
}

func (urm updateRepositoryMiddleware) RemoveByGroup(ctx context.Context, groupID string) error {
	span := dbutil.CreateSpan(ctx, urm.tracer, removeUpdatesByGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.RemoveByGroup(ctx, groupID)
}

func (urm updateRepositoryMiddleware) RetrieveThingUpdates(ctx context.Context, updateID string, pm ota.PageMetadata) (ota.ThingUpdatesPage, error) {
	span := dbutil.CreateSpan(ctx, urm.tracer, retrieveThingUpdates)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.RetrieveThingUpdates(ctx, updateID, pm)
}

func (urm updateRepositoryMiddleware) RetrieveThingUpdate(ctx context.Context, updateID, thingID string) (ota.ThingUpdate, error) {
	span := dbutil.CreateSpan(ctx, urm.tracer, retrieveThingUpdate)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.RetrieveThingUpdate(ctx, updateID, thingID)
}

func (urm updateRepositoryMiddleware) RetrieveActiveThingUpdate(ctx context.Context, thingID string) (ota.ThingUpdate, error) {
	span := dbutil.CreateSpan(ctx, urm.tracer, retrieveActiveThingUpdate)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.RetrieveActiveThingUpdate(ctx, thingID)
}

func (urm updateRepositoryMiddleware) UpdateThingUpdate(ctx context.Context, tu ota.ThingUpdate) error {
	span := dbutil.CreateSpan(ctx, urm.tracer, updateThingUpdate)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.UpdateThingUpdate(ctx, tu)
}

func (urm updateRepositoryMiddleware) RetryFailed(ctx context.Context, updateID string) ([]string, error) {
	span := dbutil.CreateSpan(ctx, urm.tracer, retryFailedThingUpdates)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.RetryFailed(ctx, updateID)
}

func (urm updateRepositoryMiddleware) FailTimedOut(ctx context.Context, now time.Time) error {
	span := dbutil.CreateSpan(ctx, urm.tracer, failTimedOutThingUpdates)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.FailTimedOut(ctx, now)
}

func (urm updateRepositoryMiddleware) RemoveThingUpdatesByThing(ctx context.Context, thingID string) error {
	span := dbutil.CreateSpan(ctx, urm.tracer, removeThingUpdatesByThing)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return urm.repo.RemoveThingUpdatesByThing(ctx, thingID)
}
