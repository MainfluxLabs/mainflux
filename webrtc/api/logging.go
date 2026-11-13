// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

//go:build !test
// +build !test

package api

import (
	"context"
	"fmt"
	"time"

	log "github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/authn"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/webrtc"
)

var _ webrtc.Service = (*loggingMiddleware)(nil)

type loggingMiddleware struct {
	logger log.Logger
	svc    webrtc.Service
}

// LoggingMiddleware adds logging facilities to the core service.
func LoggingMiddleware(svc webrtc.Service, logger log.Logger) webrtc.Service {
	return &loggingMiddleware{logger, svc}
}

func (lm *loggingMiddleware) ViewICEServers(ctx context.Context, token string) (response []webrtc.ICEServer, err error) {
	defer func(begin time.Time) {
		email := authn.EmailFromToken(token)
		message := fmt.Sprintf("Method view_ice_servers by user %s took %s to complete", email, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ViewICEServers(ctx, token)
}

func (lm *loggingMiddleware) View(ctx context.Context, token, thingID, offer string) (response webrtc.Session, err error) {
	defer func(begin time.Time) {
		email := authn.EmailFromToken(token)
		message := fmt.Sprintf("Method view for thing %s by user %s took %s to complete", thingID, email, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.View(ctx, token, thingID, offer)
}

func (lm *loggingMiddleware) AddViewerCandidate(ctx context.Context, token, sessionID string, candidate webrtc.ICECandidate) (err error) {
	defer func(begin time.Time) {
		email := authn.EmailFromToken(token)
		message := fmt.Sprintf("Method add_viewer_candidate for session %s by user %s took %s to complete", sessionID, email, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.AddViewerCandidate(ctx, token, sessionID, candidate)
}

func (lm *loggingMiddleware) ConsumeMessage(subject string, msg protomfx.Message) (err error) {
	defer func(begin time.Time) {
		message := fmt.Sprintf("Method consume_message for subject %s and thing %s took %s to complete", subject, msg.ThingId, time.Since(begin))
		if err != nil {
			lm.logger.Warn(fmt.Sprintf("%s with error: %s.", message, err))
			return
		}
		lm.logger.Info(fmt.Sprintf("%s without errors.", message))
	}(time.Now())

	return lm.svc.ConsumeMessage(subject, msg)
}
