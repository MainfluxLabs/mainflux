// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package http contains the domain concept definitions needed to support
// Mainflux http adapter service functionality.
package http

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

// Service specifies coap service API.
type Service interface {
	// Publish Messssage
	Publish(ctx context.Context, token string, msg protomfx.Message) (m protomfx.Message, err error)
}

var _ Service = (*adapterService)(nil)

type adapterService struct {
	publisher messaging.Publisher
	things    protomfx.ThingsServiceClient
}

// New instantiates the HTTP adapter implementation.
func New(publisher messaging.Publisher, things protomfx.ThingsServiceClient) Service {
	return &adapterService{
		publisher: publisher,
		things:    things,
	}
}

func (as *adapterService) Publish(ctx context.Context, key string, msg protomfx.Message) (m protomfx.Message, err error) {
	cr := &protomfx.ConnByKeyReq{
		Key: key,
	}
	conn, err := as.things.GetConnByKey(ctx, cr)
	if err != nil {
		return protomfx.Message{}, err
	}

	m = messaging.CreateMessage(conn, msg.Protocol, msg.Subtopic, &msg.Payload)

	return m, as.publisher.Publish(m)
}
