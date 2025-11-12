// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package brokers

import (
	"context"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/events"
	"github.com/MainfluxLabs/mainflux/pkg/events/redis"
)

func NewSubscriber(_ context.Context, url, stream, consumer string, logger logger.Logger) (events.Subscriber, error) {
	s, err := redis.NewSubscriber(url, stream, consumer, logger)
	if err != nil {
		return nil, err
	}

	return s, nil
}
