// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/things"
	"github.com/go-kit/kit/metrics"
)

var _ things.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     things.Service
}

// MetricsMiddleware instruments core service by tracking request count and latency.
func MetricsMiddleware(svc things.Service, counter metrics.Counter, latency metrics.Histogram) things.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (ms *metricsMiddleware) CreateThings(ctx context.Context, token string, ths ...things.Thing) (saved []things.Thing, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "create_things").Add(1)
		ms.latency.With("method", "create_things").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.CreateThings(ctx, token, ths...)
}

func (ms *metricsMiddleware) UpdateThing(ctx context.Context, token string, thing things.Thing) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_thing").Add(1)
		ms.latency.With("method", "update_thing").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UpdateThing(ctx, token, thing)
}

func (ms *metricsMiddleware) UpdateKey(ctx context.Context, token, id, key string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_key").Add(1)
		ms.latency.With("method", "update_key").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UpdateKey(ctx, token, id, key)
}

func (ms *metricsMiddleware) ViewThing(ctx context.Context, token, id string) (things.Thing, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_thing").Add(1)
		ms.latency.With("method", "view_thing").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewThing(ctx, token, id)
}

func (ms *metricsMiddleware) ListThings(ctx context.Context, token string, pm things.PageMetadata) (things.ThingsPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_things").Add(1)
		ms.latency.With("method", "list_things").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListThings(ctx, token, pm)
}

func (ms *metricsMiddleware) ListThingsByProfile(ctx context.Context, token, chID string, pm things.PageMetadata) (things.ThingsPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_things_by_profile").Add(1)
		ms.latency.With("method", "list_things_by_profile").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListThingsByProfile(ctx, token, chID, pm)
}

func (ms *metricsMiddleware) RemoveThings(ctx context.Context, token string, id ...string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_things").Add(1)
		ms.latency.With("method", "remove_things").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveThings(ctx, token, id...)
}

func (ms *metricsMiddleware) CreateProfiles(ctx context.Context, token string, profiles ...things.Profile) (saved []things.Profile, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "create_profiles").Add(1)
		ms.latency.With("method", "create_profiles").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.CreateProfiles(ctx, token, profiles...)
}

func (ms *metricsMiddleware) UpdateProfile(ctx context.Context, token string, profile things.Profile) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_profile").Add(1)
		ms.latency.With("method", "update_profile").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UpdateProfile(ctx, token, profile)
}

func (ms *metricsMiddleware) ViewProfile(ctx context.Context, token, id string) (things.Profile, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_profile").Add(1)
		ms.latency.With("method", "view_profile").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewProfile(ctx, token, id)
}

func (ms *metricsMiddleware) ListProfiles(ctx context.Context, token string, pm things.PageMetadata) (things.ProfilesPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_profiles").Add(1)
		ms.latency.With("method", "list_profiles").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListProfiles(ctx, token, pm)
}

func (ms *metricsMiddleware) ViewProfileByThing(ctx context.Context, token, thID string) (things.Profile, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_profile_by_thing").Add(1)
		ms.latency.With("method", "view_profile_by_thing").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewProfileByThing(ctx, token, thID)
}

func (ms *metricsMiddleware) RemoveProfiles(ctx context.Context, token string, ids ...string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_profiles").Add(1)
		ms.latency.With("method", "remove_profiles").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveProfiles(ctx, token, ids...)
}

func (ms *metricsMiddleware) ViewProfileConfig(ctx context.Context, chID string) (things.Config, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_profile_config").Add(1)
		ms.latency.With("method", "view_profile_config").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewProfileConfig(ctx, chID)
}

func (ms *metricsMiddleware) Connect(ctx context.Context, token, chID string, thIDs []string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "connect").Add(1)
		ms.latency.With("method", "connect").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Connect(ctx, token, chID, thIDs)
}

func (ms *metricsMiddleware) Disconnect(ctx context.Context, token, chID string, thIDs []string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "disconnect").Add(1)
		ms.latency.With("method", "disconnect").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Disconnect(ctx, token, chID, thIDs)
}

func (ms *metricsMiddleware) GetConnByKey(ctx context.Context, key string) (things.Connection, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "get_conn_by_key").Add(1)
		ms.latency.With("method", "get_conn_by_key").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.GetConnByKey(ctx, key)
}

func (ms *metricsMiddleware) Authorize(ctx context.Context, ar things.AuthorizeReq) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "authorize").Add(1)
		ms.latency.With("method", "authorize").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Authorize(ctx, ar)
}

func (ms *metricsMiddleware) Identify(ctx context.Context, key string) (string, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "identify").Add(1)
		ms.latency.With("method", "identify").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Identify(ctx, key)
}

func (ms *metricsMiddleware) GetConfigByThingID(ctx context.Context, thingID string) (things.Config, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "get_config_by_thing_id").Add(1)
		ms.latency.With("method", "get_config_by_thing_id").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.GetConfigByThingID(ctx, thingID)
}

func (ms *metricsMiddleware) GetGroupIDByThingID(ctx context.Context, thingID string) (string, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "get_group_id_by_thing_id").Add(1)
		ms.latency.With("method", "get_group_id_by_thing_id").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.GetGroupIDByThingID(ctx, thingID)
}

func (ms *metricsMiddleware) Backup(ctx context.Context, token string) (bk things.Backup, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "backup").Add(1)
		ms.latency.With("method", "backup").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Backup(ctx, token)
}

func (ms *metricsMiddleware) Restore(ctx context.Context, token string, backup things.Backup) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "restore").Add(1)
		ms.latency.With("method", "restore").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.Restore(ctx, token, backup)
}

func (ms *metricsMiddleware) CreateGroups(ctx context.Context, token string, grs ...things.Group) ([]things.Group, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "create_groups").Add(1)
		ms.latency.With("method", "create_groups").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.CreateGroups(ctx, token, grs...)
}

func (ms *metricsMiddleware) UpdateGroup(ctx context.Context, token string, g things.Group) (things.Group, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_group").Add(1)
		ms.latency.With("method", "update_group").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UpdateGroup(ctx, token, g)
}

func (ms *metricsMiddleware) ViewGroup(ctx context.Context, token, id string) (things.Group, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_group").Add(1)
		ms.latency.With("method", "view_group").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewGroup(ctx, token, id)
}

func (ms *metricsMiddleware) ListGroups(ctx context.Context, token, orgID string, pm things.PageMetadata) (things.GroupPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_group").Add(1)
		ms.latency.With("method", "list_group").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListGroups(ctx, token, orgID, pm)
}

func (ms *metricsMiddleware) ListGroupsByIDs(ctx context.Context, groupIDs []string) ([]things.Group, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_groups_by_ids").Add(1)
		ms.latency.With("method", "list_groups_by_ids").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListGroupsByIDs(ctx, groupIDs)
}

func (ms *metricsMiddleware) ListThingsByGroup(ctx context.Context, token, groupID string, pm things.PageMetadata) (tp things.ThingsPage, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_things_by_group").Add(1)
		ms.latency.With("method", "list_things_by_group").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListThingsByGroup(ctx, token, groupID, pm)
}

func (ms *metricsMiddleware) ViewGroupByThing(ctx context.Context, token, thingID string) (things.Group, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_group_by_thing").Add(1)
		ms.latency.With("method", "view_group_by_thing").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewGroupByThing(ctx, token, thingID)
}

func (ms *metricsMiddleware) RemoveGroups(ctx context.Context, token string, ids ...string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_groups").Add(1)
		ms.latency.With("method", "remove_groups").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveGroups(ctx, token, ids...)
}

func (ms *metricsMiddleware) ListProfilesByGroup(ctx context.Context, token, groupID string, pm things.PageMetadata) (things.ProfilesPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_profiles_by_group").Add(1)
		ms.latency.With("method", "list_profiles_by_group").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListProfilesByGroup(ctx, token, groupID, pm)
}

func (ms *metricsMiddleware) ViewGroupByProfile(ctx context.Context, token, profileID string) (things.Group, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_group_by_profile").Add(1)
		ms.latency.With("method", "view_group_by_profile").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewGroupByProfile(ctx, token, profileID)
}

func (ms *metricsMiddleware) CreateRolesByGroup(ctx context.Context, token string, gms ...things.GroupMember) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "create_roles_by_group").Add(1)
		ms.latency.With("method", "create_roles_by_group").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.CreateRolesByGroup(ctx, token, gms...)
}

func (ms *metricsMiddleware) ListRolesByGroup(ctx context.Context, token, groupID string, pm things.PageMetadata) (things.GroupMembersPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_roles_by_group").Add(1)
		ms.latency.With("method", "list_roles_by_group").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListRolesByGroup(ctx, token, groupID, pm)
}

func (ms *metricsMiddleware) UpdateRolesByGroup(ctx context.Context, token string, gms ...things.GroupMember) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_roles_by_group").Add(1)
		ms.latency.With("method", "update_roles_by_group").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UpdateRolesByGroup(ctx, token, gms...)
}

func (ms *metricsMiddleware) RemoveRolesByGroup(ctx context.Context, token, groupID string, memberIDs ...string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_roles_by_group").Add(1)
		ms.latency.With("method", "remove_roles_by_group").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveRolesByGroup(ctx, token, groupID, memberIDs...)
}
