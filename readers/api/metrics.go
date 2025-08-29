// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test

package api

import (
	"context"
	"time"

	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/go-kit/kit/metrics"
)

var _ readers.MessageRepository = (*metricsMiddleware)(nil)

type metricsMiddleware struct {
	counter metrics.Counter
	latency metrics.Histogram
	svc     readers.MessageRepository
}

// MetricsMiddleware instruments core service by tracking request count and latency.
func MetricsMiddleware(svc readers.MessageRepository, counter metrics.Counter, latency metrics.Histogram) readers.MessageRepository {
	return &metricsMiddleware{
		counter: counter,
		latency: latency,
		svc:     svc,
	}
}

//	func (mm *metricsMiddleware) ListAllMessages(rpm readers.PageMetadata) (readers.MessagesPage, error) {
//		defer func(begin time.Time) {
//			mm.counter.With("method", "list_all_messages").Add(1)
//			mm.latency.With("method", "list_all_messages").Observe(time.Since(begin).Seconds())
//		}(time.Now())
//
//		return mm.svc.ListAllMessages(rpm)
//	}
//
//	func (mm *metricsMiddleware) DeleteMessages(ctx context.Context, rpm readers.PageMetadata) error {
//		defer func(begin time.Time) {
//			mm.counter.With("method", "delete_messages").Add(1)
//			mm.latency.With("method", "delete_messages").Observe(time.Since(begin).Seconds())
//		}(time.Now())
//
//		return mm.svc.DeleteMessages(ctx, rpm)
//	}
//
//	func (mm *metricsMiddleware) Backup(rpm readers.PageMetadata) (readers.MessagesPage, error) {
//		defer func(begin time.Time) {
//			mm.counter.With("method", "backup").Add(1)
//			mm.latency.With("method", "backup").Observe(time.Since(begin).Seconds())
//		}(time.Now())
//
//		return mm.svc.Backup(rpm)
//	}
//
//	func (mm *metricsMiddleware) Restore(ctx context.Context, format string, messages ...readers.Message) error {
//		defer func(begin time.Time) {
//			mm.counter.With("method", "restore").Add(1)
//			mm.latency.With("method", "restore").Observe(time.Since(begin).Seconds())
//		}(time.Now())
//
//		return mm.svc.Restore(ctx, format, messages...)
//	}
func (mm *metricsMiddleware) ListJSONMessages(rpm readers.PageMetadata) (readers.MessagesPage, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "list_json_messages").Add(1)
		mm.latency.With("method", "list_json_messages").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.ListJSONMessages(rpm)
}

func (mm *metricsMiddleware) ListSenMLMessages(rpm readers.PageMetadata) (readers.MessagesPage, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "list_senml_messages").Add(1)
		mm.latency.With("method", "list_senml_messages").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.ListSenMLMessages(rpm)
}

func (mm *metricsMiddleware) BackupJSONMessages(rpm readers.PageMetadata) (readers.MessagesPage, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "backup_json_messages").Add(1)
		mm.latency.With("method", "backup_json_messages").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.BackupJSONMessages(rpm)

}

func (mm *metricsMiddleware) BackupSenMLMessages(rpm readers.PageMetadata) (readers.MessagesPage, error) {
	defer func(begin time.Time) {
		mm.counter.With("method", "backup_senml_messages").Add(1)
		mm.latency.With("method", "backup_senml_messages").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.BackupSenMLMessages(rpm)
}

func (mm *metricsMiddleware) RestoreJSONMessages(ctx context.Context, messages ...readers.Message) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "restore_json_messages").Add(1)
		mm.latency.With("method", "restore_json_messages").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.RestoreJSONMessages(ctx, messages...)
}

func (mm *metricsMiddleware) RestoreSenMLMessageS(ctx context.Context, messages ...readers.Message) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "restore_senml_messages").Add(1)
		mm.latency.With("method", "restore_senml_messages").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.RestoreSenMLMessageS(ctx, messages...)
}

func (mm *metricsMiddleware) DeleteJSONMessages(ctx context.Context, rpm readers.PageMetadata) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "delete_json_messages").Add(1)
		mm.latency.With("method", "delete_json_messages").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.DeleteJSONMessages(ctx, rpm)
}

func (mm *metricsMiddleware) DeleteSenMLMessages(ctx context.Context, rpm readers.PageMetadata) error {
	defer func(begin time.Time) {
		mm.counter.With("method", "delete_json_messages").Add(1)
		mm.latency.With("method", "delete_json_messages").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mm.svc.DeleteSenMLMessages(ctx, rpm)
}
