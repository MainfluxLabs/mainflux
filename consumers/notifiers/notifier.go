// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package notifiers

import (
	"context"
	"errors"

	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/things"
)

// ErrNotify wraps sending notification errors,
var ErrNotify = errors.New("failed to send notification")

// Notifier represents an API for sending notification.
type Notifier interface {
	// Notify method is used to send notification for the
	// received message to the provided list of receivers.
	Notify(from string, to []string, msg protomfx.Message) error
}

// NotifierRepository specifies a notifier persistence API.
type NotifierRepository interface {
	// Save persists multiple notifiers. Notifiers are saved using a transaction.
	// If one notifier fails then none will be saved.
	// Successful operation is indicated by non-nil error response.
	Save(ctx context.Context, nfs ...things.Notifier) ([]things.Notifier, error)

	// RetrieveByGroupID retrieves notifiers related to
	// a certain group identified by a given ID.
	RetrieveByGroupID(ctx context.Context, groupID string) ([]things.Notifier, error)

	// RetrieveByID retrieves the notifier having the provided identifier
	RetrieveByID(ctx context.Context, id string) (things.Notifier, error)

	// Update performs an update to the existing notifier. A non-nil error is
	// returned to indicate operation failure.
	Update(ctx context.Context, n things.Notifier) error

	// Remove removes the notifiers having the provided identifiers
	Remove(ctx context.Context, groupID string, ids ...string) error
}
