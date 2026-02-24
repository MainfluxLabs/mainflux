// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test
// +build !test

package api

import (
	"context"
	"io"
	"time"

	"github.com/MainfluxLabs/mainflux/filestore"
	"github.com/go-kit/kit/metrics"
)

var _ filestore.Service = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     filestore.Service
}

// MetricsMiddleware instruments core service by tracking request count and
// latency.
func MetricsMiddleware(svc filestore.Service, counter metrics.Counter, latency metrics.Histogram) filestore.Service {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

func (ms *metricsMiddleware) SaveFile(ctx context.Context, file io.Reader, key string, fi filestore.FileInfo) (err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "save_file").Add(1)
		ms.latency.With("method", "save_file").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.SaveFile(ctx, file, key, fi)
}

func (ms *metricsMiddleware) UpdateFile(ctx context.Context, key string, fi filestore.FileInfo) (err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_file").Add(1)
		ms.latency.With("method", "update_file").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UpdateFile(ctx, key, fi)
}

func (ms *metricsMiddleware) ViewFile(ctx context.Context, key string, fi filestore.FileInfo) (data []byte, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_file").Add(1)
		ms.latency.With("method", "view_file").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewFile(ctx, key, fi)
}

func (ms *metricsMiddleware) ListFiles(ctx context.Context, key string, fi filestore.FileInfo, pm filestore.PageMetadata) (files filestore.FileThingsPage, err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_files").Add(1)
		ms.latency.With("method", "list_files").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListFiles(ctx, key, fi, pm)
}

func (ms *metricsMiddleware) RemoveFile(ctx context.Context, key string, fi filestore.FileInfo) (err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_file").Add(1)
		ms.latency.With("method", "remove_file").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveFile(ctx, key, fi)
}

func (ms *metricsMiddleware) RemoveFiles(ctx context.Context, thingID string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_files").Add(1)
		ms.latency.With("method", "remove_files").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveFiles(ctx, thingID)
}

func (ms *metricsMiddleware) SaveGroupFile(ctx context.Context, file io.Reader, token, groupID string, fi filestore.FileInfo) (err error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "save_group_file").Add(1)
		ms.latency.With("method", "save_group_file").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.SaveGroupFile(ctx, file, token, groupID, fi)
}

func (ms *metricsMiddleware) UpdateGroupFile(ctx context.Context, token, groupID string, fi filestore.FileInfo) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "update_group_file").Add(1)
		ms.latency.With("method", "update_group_file").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.UpdateGroupFile(ctx, token, groupID, fi)
}

func (ms *metricsMiddleware) ViewGroupFile(ctx context.Context, token, groupID string, fi filestore.FileInfo) ([]byte, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_group_file").Add(1)
		ms.latency.With("method", "view_group_file").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewGroupFile(ctx, token, groupID, fi)
}

func (ms *metricsMiddleware) ListGroupFiles(ctx context.Context, token, groupID string, fi filestore.FileInfo, pm filestore.PageMetadata) (filestore.FileGroupsPage, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "list_group_files").Add(1)
		ms.latency.With("method", "list_group_files").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ListGroupFiles(ctx, token, groupID, fi, pm)
}

func (ms *metricsMiddleware) RemoveGroupFile(ctx context.Context, token, groupID string, fi filestore.FileInfo) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_group_file").Add(1)
		ms.latency.With("method", "remove_group_file").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveGroupFile(ctx, token, groupID, fi)
}

func (ms *metricsMiddleware) RemoveAllFilesByGroup(ctx context.Context, groupID string) error {
	defer func(begin time.Time) {
		ms.counter.With("method", "remove_all_files_by_group").Add(1)
		ms.latency.With("method", "remove_all_files_by_group").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.RemoveAllFilesByGroup(ctx, groupID)
}

func (ms *metricsMiddleware) ViewGroupFileByKey(ctx context.Context, thingKey string, fi filestore.FileInfo) ([]byte, error) {
	defer func(begin time.Time) {
		ms.counter.With("method", "view_group_file_by_key").Add(1)
		ms.latency.With("method", "view_group_file_by_key").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return ms.svc.ViewGroupFileByKey(ctx, thingKey, fi)
}
