// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/things"
	"github.com/opentracing/opentracing-go"
)

const (
	saveProfilesOp               = "save_profiles"
	updateProfileOp              = "update_profile"
	retrieveProfileByIDOp        = "retrieve_profile_by_id"
	retrieveByThingOp            = "retrieve_by_thing"
	retrieveProfilesByGroupIDsOp = "retrieve_profiles_by_group_ids"
	removeProfileOp              = "retrieve_profile"
	connectOp                    = "connect"
	disconnectOp                 = "disconnect"
	hasThingOp                   = "has_thing"
	retrieveAllProfilesOp        = "retrieve_all_profiles"
	retrieveAllConnectionsOp     = "retrieve_all_connections"
)

var (
	_ things.ProfileRepository = (*profileRepositoryMiddleware)(nil)
	_ things.ProfileCache      = (*profileCacheMiddleware)(nil)
)

type profileRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   things.ProfileRepository
}

// ProfileRepositoryMiddleware tracks request and their latency, and adds spans
// to context.
func ProfileRepositoryMiddleware(tracer opentracing.Tracer, repo things.ProfileRepository) things.ProfileRepository {
	return profileRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (crm profileRepositoryMiddleware) Save(ctx context.Context, profiles ...things.Profile) ([]things.Profile, error) {
	span := createSpan(ctx, crm.tracer, saveProfilesOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.Save(ctx, profiles...)
}

func (crm profileRepositoryMiddleware) Update(ctx context.Context, ch things.Profile) error {
	span := createSpan(ctx, crm.tracer, updateProfileOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.Update(ctx, ch)
}

func (crm profileRepositoryMiddleware) RetrieveByID(ctx context.Context, id string) (things.Profile, error) {
	span := createSpan(ctx, crm.tracer, retrieveProfileByIDOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RetrieveByID(ctx, id)
}

func (crm profileRepositoryMiddleware) RetrieveByGroupIDs(ctx context.Context, ids []string, pm things.PageMetadata) (things.ProfilesPage, error) {
	span := createSpan(ctx, crm.tracer, retrieveProfilesByGroupIDsOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RetrieveByGroupIDs(ctx, ids, pm)
}

func (crm profileRepositoryMiddleware) RetrieveByThing(ctx context.Context, thID string) (things.Profile, error) {
	span := createSpan(ctx, crm.tracer, retrieveByThingOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RetrieveByThing(ctx, thID)
}

func (crm profileRepositoryMiddleware) Remove(ctx context.Context, ids ...string) error {
	span := createSpan(ctx, crm.tracer, removeProfileOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.Remove(ctx, ids...)
}

func (crm profileRepositoryMiddleware) Connect(ctx context.Context, chID string, thIDs []string) error {
	span := createSpan(ctx, crm.tracer, connectOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.Connect(ctx, chID, thIDs)
}

func (crm profileRepositoryMiddleware) Disconnect(ctx context.Context, chID string, thIDs []string) error {
	span := createSpan(ctx, crm.tracer, disconnectOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.Disconnect(ctx, chID, thIDs)
}

func (crm profileRepositoryMiddleware) RetrieveConnByThingKey(ctx context.Context, key string) (things.Connection, error) {
	span := createSpan(ctx, crm.tracer, hasThingOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RetrieveConnByThingKey(ctx, key)
}

func (crm profileRepositoryMiddleware) RetrieveAll(ctx context.Context) ([]things.Profile, error) {
	span := createSpan(ctx, crm.tracer, retrieveAllProfilesOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RetrieveAll(ctx)
}

func (crm profileRepositoryMiddleware) RetrieveByAdmin(ctx context.Context, pm things.PageMetadata) (things.ProfilesPage, error) {
	span := createSpan(ctx, crm.tracer, retrieveAllProfilesOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RetrieveByAdmin(ctx, pm)
}

func (crm profileRepositoryMiddleware) RetrieveAllConnections(ctx context.Context) ([]things.Connection, error) {
	span := createSpan(ctx, crm.tracer, retrieveAllConnectionsOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RetrieveAllConnections(ctx)
}

type profileCacheMiddleware struct {
	tracer opentracing.Tracer
	cache  things.ProfileCache
}

// ProfileCacheMiddleware tracks request and their latency, and adds spans
// to context.
func ProfileCacheMiddleware(tracer opentracing.Tracer, cache things.ProfileCache) things.ProfileCache {
	return profileCacheMiddleware{
		tracer: tracer,
		cache:  cache,
	}
}

func (ccm profileCacheMiddleware) Connect(ctx context.Context, profileID, thingID string) error {
	span := createSpan(ctx, ccm.tracer, connectOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return ccm.cache.Connect(ctx, profileID, thingID)
}

func (ccm profileCacheMiddleware) HasThing(ctx context.Context, profileID, thingID string) bool {
	span := createSpan(ctx, ccm.tracer, hasThingOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return ccm.cache.HasThing(ctx, profileID, thingID)
}

func (ccm profileCacheMiddleware) Disconnect(ctx context.Context, profileID, thingID string) error {
	span := createSpan(ctx, ccm.tracer, disconnectOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return ccm.cache.Disconnect(ctx, profileID, thingID)
}

func (ccm profileCacheMiddleware) Remove(ctx context.Context, profileID string) error {
	span := createSpan(ctx, ccm.tracer, removeProfileOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return ccm.cache.Remove(ctx, profileID)
}
