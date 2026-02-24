// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test
// +build !test

package api

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/MainfluxLabs/mainflux/filestore"
	log "github.com/MainfluxLabs/mainflux/logger"
)

var _ filestore.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    filestore.Service
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc filestore.Service, logger log.Logger) filestore.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) SaveFile(ctx context.Context, file io.Reader, key string, fi filestore.FileInfo) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method save_file took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.SaveFile(ctx, file, key, fi)
}

func (lm *loggingMiddleware) UpdateFile(ctx context.Context, key string, fi filestore.FileInfo) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_file took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateFile(ctx, key, fi)
}

func (lm *loggingMiddleware) ViewFile(ctx context.Context, key string, fi filestore.FileInfo) (data []byte, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_file took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewFile(ctx, key, fi)
}

func (lm *loggingMiddleware) ListFiles(ctx context.Context, key string, fi filestore.FileInfo, pm filestore.PageMetadata) (files filestore.FileThingsPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_files took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListFiles(ctx, key, fi, pm)
}

func (lm *loggingMiddleware) RemoveFile(ctx context.Context, key string, fi filestore.FileInfo) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_file took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveFile(ctx, key, fi)
}

func (lm *loggingMiddleware) RemoveFiles(ctx context.Context, thingID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_files for thing %s took %s to complete", thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveFiles(ctx, thingID)
}

func (lm *loggingMiddleware) SaveGroupFile(ctx context.Context, file io.Reader, token, groupID string, fi filestore.FileInfo) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method save_group_file took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.SaveGroupFile(ctx, file, token, groupID, fi)
}

func (lm *loggingMiddleware) UpdateGroupFile(ctx context.Context, token, groupID string, fi filestore.FileInfo) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_group_file took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateGroupFile(ctx, token, groupID, fi)
}

func (lm *loggingMiddleware) ViewGroupFile(ctx context.Context, token, groupID string, fi filestore.FileInfo) (data []byte, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_group_file for file name %s took %s to complete", fi.Name, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewGroupFile(ctx, token, groupID, fi)
}

func (lm *loggingMiddleware) ListGroupFiles(ctx context.Context, token, groupID string, fi filestore.FileInfo, pm filestore.PageMetadata) (files filestore.FileGroupsPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_group_files took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListGroupFiles(ctx, token, groupID, fi, pm)
}

func (lm *loggingMiddleware) RemoveGroupFile(ctx context.Context, token, groupID string, fi filestore.FileInfo) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_group_file took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveGroupFile(ctx, token, groupID, fi)
}

func (lm *loggingMiddleware) RemoveAllFilesByGroup(ctx context.Context, groupID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_all_files_by_group for id %s took %s to complete", groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveAllFilesByGroup(ctx, groupID)
}

func (lm *loggingMiddleware) ViewGroupFileByKey(ctx context.Context, thingKey string, fi filestore.FileInfo) (data []byte, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_group_file_by_key for file name %s took %s to complete", fi.Name, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewGroupFileByKey(ctx, thingKey, fi)
}
