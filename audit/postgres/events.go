// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"github.com/MainfluxLabs/mainflux/audit"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
)

var _ audit.EventRepository = (*eventRepository)(nil)

type eventRepository struct {
	db dbutil.Database
}

func NewEventRepository(db dbutil.Database) audit.EventRepository {
	return &eventRepository{db: db}
}
