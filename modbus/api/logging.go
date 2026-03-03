// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"fmt"
	"time"

	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/modbus"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

var _ modbus.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    modbus.Service
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc modbus.Service, logger log.Logger) modbus.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) CreateClients(ctx context.Context, token, thingID string, clients ...modbus.Client) (response []modbus.Client, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method create_clients for clients %v took %s to complete", response, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.CreateClients(ctx, token, thingID, clients...)
}

func (lm *loggingMiddleware) ListClientsByThing(ctx context.Context, token, thingID string, pm apiutil.PageMetadata) (response modbus.ClientsPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_clients_by_thing for id %s took %s to complete", thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListClientsByThing(ctx, token, thingID, pm)
}

func (lm *loggingMiddleware) ListClientsByGroup(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (response modbus.ClientsPage, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method list_clients_by_group for id %s took %s to complete", groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ListClientsByGroup(ctx, token, groupID, pm)
}

func (lm *loggingMiddleware) ViewClient(ctx context.Context, token, id string) (response modbus.Client, err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method view_client for id %s took %s to complete", id, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewClient(ctx, token, id)
}

func (lm *loggingMiddleware) UpdateClient(ctx context.Context, token string, client modbus.Client) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method update_client for id %s took %s to complete", client.ID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.UpdateClient(ctx, token, client)
}

func (lm *loggingMiddleware) RemoveClients(ctx context.Context, token string, id ...string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_clients took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveClients(ctx, token, id...)
}

func (lm *loggingMiddleware) RemoveClientsByThing(ctx context.Context, thingID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_clients_by_thing for id %s took %s to complete", thingID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveClientsByThing(ctx, thingID)
}

func (lm *loggingMiddleware) RemoveClientsByGroup(ctx context.Context, groupID string) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method remove_clients_by_group for id %s took %s to complete", groupID, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RemoveClientsByGroup(ctx, groupID)
}

func (lm *loggingMiddleware) RescheduleTasks(ctx context.Context, profileID string, config map[string]any) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method reschedule_tasks for profile %s and config %v took %s to complete", profileID, config, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.RescheduleTasks(ctx, profileID, config)
}

func (lm *loggingMiddleware) LoadAndScheduleTasks(ctx context.Context) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method load_and_schedule_tasks took %s to complete", time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.LoadAndScheduleTasks(ctx)
}
