// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package http contains the domain concept definitions needed to support
// Mainflux http adapter service functionality.
package http

import (
	"context"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
)

const senmlContentType = "application/senml+json"

// Service specifies coap service API.
type Service interface {
	// Publish Messssage
	Publish(ctx context.Context, token string, msg messaging.Message) error
}

var _ Service = (*adapterService)(nil)

type adapterService struct {
	publisher messaging.Publisher
	things    mainflux.ThingsServiceClient
}

// New instantiates the HTTP adapter implementation.
func New(publisher messaging.Publisher, things mainflux.ThingsServiceClient) Service {
	return &adapterService{
		publisher: publisher,
		things:    things,
	}
}

func (as *adapterService) Publish(ctx context.Context, key string, msg messaging.Message) error {
	cr := &mainflux.ConnByKeyReq{
		Key: key,
	}
	conn, err := as.things.GetConnByKey(ctx, cr)
	if err != nil {
		return err
	}
	msg.Publisher = conn.ThingID

	switch {
	case conn.Profile != nil:
		msg.Profile = &messaging.Profile{
			ContentType: conn.Profile.ContentType,
			TimeField: &messaging.TimeField{
				Name:     conn.Profile.TimeField.Name,
				Format:   conn.Profile.TimeField.Format,
				Location: conn.Profile.TimeField.Location,
			},
		}

	default:
		msg.Profile = &messaging.Profile{
			ContentType: senmlContentType,
		}
	}

	return as.publisher.Publish(conn.ChannelID, msg)
}
