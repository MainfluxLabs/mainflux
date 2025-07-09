// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/opentracing/opentracing-go"
)

const (
	saveGroupIDByProfileID     = "save_group_id_by_profile_id"
	saveProfiles               = "save_profiles"
	updateProfile              = "update_profile"
	retrieveProfileByID        = "retrieve_profile_by_id"
	retrieveByThing            = "retrieve_by_thing"
	retrieveProfilesByGroupIDs = "retrieve_profiles_by_group_ids"
	removeProfile              = "remove_profile"
	removeGroupIDByProfileID   = "remove_group_id_by_profile_id"
	retrieveAllProfiles        = "retrieve_all_profiles"
	retrieveGroupIDByProfileID = "retrieve_group_id_by_profile_id"
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
	span := createSpan(ctx, crm.tracer, saveProfiles)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.Save(ctx, profiles...)
}

func (crm profileRepositoryMiddleware) Update(ctx context.Context, pr things.Profile) error {
	span := createSpan(ctx, crm.tracer, updateProfile)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.Update(ctx, pr)
}

func (crm profileRepositoryMiddleware) RetrieveByID(ctx context.Context, id string) (things.Profile, error) {
	span := createSpan(ctx, crm.tracer, retrieveProfileByID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RetrieveByID(ctx, id)
}

func (crm profileRepositoryMiddleware) RetrieveByGroupIDs(ctx context.Context, ids []string, pm apiutil.PageMetadata) (things.ProfilesPage, error) {
	span := createSpan(ctx, crm.tracer, retrieveProfilesByGroupIDs)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RetrieveByGroupIDs(ctx, ids, pm)
}

func (crm profileRepositoryMiddleware) RetrieveByThing(ctx context.Context, thID string) (things.Profile, error) {
	span := createSpan(ctx, crm.tracer, retrieveByThing)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RetrieveByThing(ctx, thID)
}

func (crm profileRepositoryMiddleware) Remove(ctx context.Context, ids ...string) error {
	span := createSpan(ctx, crm.tracer, removeProfile)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.Remove(ctx, ids...)
}

func (crm profileRepositoryMiddleware) BackupAll(ctx context.Context) ([]things.Profile, error) {
	span := createSpan(ctx, crm.tracer, backupAll)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.BackupAll(ctx)
}

func (crm profileRepositoryMiddleware) RetrieveAll(ctx context.Context, pm apiutil.PageMetadata) (things.ProfilesPage, error) {
	span := createSpan(ctx, crm.tracer, retrieveAllProfiles)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return crm.repo.RetrieveAll(ctx, pm)
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
	span := createSpan(ctx, ccm.tracer, saveGroupIDByProfileID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return ccm.cache.SaveGroup(ctx, profileID, groupID)
}

func (ccm profileCacheMiddleware) ViewGroup(ctx context.Context, profileID string) (string, error) {
	span := createSpan(ctx, ccm.tracer, retrieveGroupIDByProfileID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return ccm.cache.ViewGroup(ctx, profileID)
}

func (ccm profileCacheMiddleware) RemoveGroup(ctx context.Context, profileID string) error {
	span := createSpan(ctx, ccm.tracer, removeGroupIDByProfileID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return ccm.cache.RemoveGroup(ctx, profileID)
}
