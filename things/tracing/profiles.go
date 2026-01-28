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
	saveGroupIDByProfileID     = "save_group_id_by_profile_id"
	saveProfiles               = "save_profiles"
	updateProfile              = "update_profile"
	retrieveProfileByID        = "retrieve_profile_by_id"
	retrieveProfileByThing     = "retrieve_profile_by_thing"
	retrieveProfilesByGroups   = "retrieve_profiles_by_groups"
	removeProfile              = "remove_profile"
	removeGroupIDByProfileID   = "remove_group_id_by_profile_id"
	retrieveAllProfiles        = "retrieve_all_profiles"
	backupAllProfiles          = "backup_all_profiles"
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

func (prm profileRepositoryMiddleware) Save(ctx context.Context, profiles ...things.Profile) ([]things.Profile, error) {
	span := dbutil.CreateSpan(ctx, prm.tracer, saveProfiles)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return prm.repo.Save(ctx, profiles...)
}

func (prm profileRepositoryMiddleware) Update(ctx context.Context, pr things.Profile) error {
	span := dbutil.CreateSpan(ctx, prm.tracer, updateProfile)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return prm.repo.Update(ctx, pr)
}

func (prm profileRepositoryMiddleware) RetrieveByID(ctx context.Context, id string) (things.Profile, error) {
	span := dbutil.CreateSpan(ctx, prm.tracer, retrieveProfileByID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return prm.repo.RetrieveByID(ctx, id)
}

func (prm profileRepositoryMiddleware) RetrieveByGroups(ctx context.Context, ids []string, pm apiutil.PageMetadata) (things.ProfilesPage, error) {
	span := dbutil.CreateSpan(ctx, prm.tracer, retrieveProfilesByGroups)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return prm.repo.RetrieveByGroups(ctx, ids, pm)
}

func (prm profileRepositoryMiddleware) RetrieveByThing(ctx context.Context, thID string) (things.Profile, error) {
	span := dbutil.CreateSpan(ctx, prm.tracer, retrieveProfileByThing)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return prm.repo.RetrieveByThing(ctx, thID)
}

func (prm profileRepositoryMiddleware) Remove(ctx context.Context, ids ...string) error {
	span := dbutil.CreateSpan(ctx, prm.tracer, removeProfile)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return prm.repo.Remove(ctx, ids...)
}

func (prm profileRepositoryMiddleware) BackupAll(ctx context.Context) ([]things.Profile, error) {
	span := dbutil.CreateSpan(ctx, prm.tracer, backupAllProfiles)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return prm.repo.BackupAll(ctx)
}

func (prm profileRepositoryMiddleware) RetrieveAll(ctx context.Context, pm apiutil.PageMetadata) (things.ProfilesPage, error) {
	span := dbutil.CreateSpan(ctx, prm.tracer, retrieveAllProfiles)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return prm.repo.RetrieveAll(ctx, pm)
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

func (pcm profileCacheMiddleware) SaveGroup(ctx context.Context, profileID, groupID string) error {
	span := dbutil.CreateSpan(ctx, pcm.tracer, saveGroupIDByProfileID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return pcm.cache.SaveGroup(ctx, profileID, groupID)
}

func (pcm profileCacheMiddleware) ViewGroup(ctx context.Context, profileID string) (string, error) {
	span := dbutil.CreateSpan(ctx, pcm.tracer, retrieveGroupIDByProfileID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return pcm.cache.ViewGroup(ctx, profileID)
}

func (pcm profileCacheMiddleware) RemoveGroup(ctx context.Context, profileID string) error {
	span := dbutil.CreateSpan(ctx, pcm.tracer, removeGroupIDByProfileID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return pcm.cache.RemoveGroup(ctx, profileID)
}
