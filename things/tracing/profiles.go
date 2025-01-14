// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/things"
	"github.com/opentracing/opentracing-go"
)

const (
	saveGroupIDByProfileIDOp     = "save_group_id_by_profile_id"
	saveProfilesOp               = "save_profiles"
	updateProfileOp              = "update_profile"
	retrieveProfileByIDOp        = "retrieve_profile_by_id"
	retrieveByThingOp            = "retrieve_by_thing"
	retrieveProfilesByGroupIDsOp = "retrieve_profiles_by_group_ids"
	removeProfileOp              = "remove_profile"
	removeGroupIDByProfileIDOp   = "remove_group_id_by_profile_id"
	retrieveAllProfilesOp        = "retrieve_all_profiles"
	retrieveGroupIDByProfileIDOp = "retrieve_group_id_by_profile_id"
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

func (crm profileRepositoryMiddleware) Update(ctx context.Context, pr things.Profile) error {
	span := createSpan(ctx, crm.tracer, updateProfileOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.Update(ctx, pr)
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

func (ccm profileCacheMiddleware) SaveGroup(ctx context.Context, profileID, groupID string) error {
	span := createSpan(ctx, ccm.tracer, saveGroupIDByProfileIDOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return ccm.cache.SaveGroup(ctx, profileID, groupID)
}

func (ccm profileCacheMiddleware) ViewGroup(ctx context.Context, profileID string) (string, error) {
	span := createSpan(ctx, ccm.tracer, retrieveGroupIDByProfileIDOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return ccm.cache.ViewGroup(ctx, profileID)
}

func (ccm profileCacheMiddleware) RemoveGroup(ctx context.Context, profileID string) error {
	span := createSpan(ctx, ccm.tracer, removeGroupIDByProfileIDOp)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return ccm.cache.RemoveGroup(ctx, profileID)
}
