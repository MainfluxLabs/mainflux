// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
)

var _ domain.ReadersClient = (*readersClient)(nil)

type readersClient struct{}

// NewReadersClient returns a no-op ReadersClient mock.
func NewReadersClient() domain.ReadersClient {
	return &readersClient{}
}

func (r *readersClient) ListJSONMessages(_ context.Context, _ domain.ThingKey, _ domain.JSONPageMetadata) (domain.JSONMessagesPage, error) {
	return domain.JSONMessagesPage{}, nil
}

func (r *readersClient) ListSenMLMessages(_ context.Context, _ domain.ThingKey, _ domain.SenMLPageMetadata) (domain.SenMLMessagesPage, error) {
	return domain.SenMLMessagesPage{}, nil
}
