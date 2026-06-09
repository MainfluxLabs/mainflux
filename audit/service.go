// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package audit

import (
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
)

type Service interface {
}

type EventRepository interface {
}

var _ Service = (*auditService)(nil)

type auditService struct {
	events EventRepository
	auth   domain.AuthClient
	things domain.ThingsClient
	idp    uuid.IDProvider
}

func New(events EventRepository, auth domain.AuthClient, things domain.ThingsClient, idp uuid.IDProvider) Service {
	return &auditService{
		events: events,
		auth:   auth,
		things: things,
		idp:    idp,
	}
}
