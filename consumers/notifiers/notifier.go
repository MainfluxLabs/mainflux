// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package notifiers

import (
	"context"
	"errors"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

// ErrNotify wraps sending notification errors,
var ErrNotify = errors.New("failed to send notification")

type Notifier struct {
	ID       string
	GroupID  string
	Name     string
	Contacts []string
	Metadata map[string]any
}

type NotifiersPage struct {
	Total     uint64
	Notifiers []Notifier
}

// Sender represents an API for sending notification.
type Sender interface {
	// Send method is used to send notification for the
	// received message to the provided list of receivers.
	Send(to []string, msg protomfx.Message) error

	// ValidateContacts method is used to validate contacts
	// to which notifications will be sent.
	ValidateContacts(contacts []string) error
}

// NotifierRepository specifies a notifier persistence API.
type NotifierRepository interface {
	// Save persists multiple notifiers. Notifiers are saved using a transaction.
	// If one notifier fails then none will be saved.
	// Successful operation is indicated by non-nil error response.
	Save(ctx context.Context, nfs ...Notifier) ([]Notifier, error)

	// RetrieveByGroup retrieves notifiers related to a certain group identified by a given ID.
	RetrieveByGroup(ctx context.Context, groupID string, pm apiutil.PageMetadata) (NotifiersPage, error)

	// RetrieveByID retrieves the notifier having the provided identifier
	RetrieveByID(ctx context.Context, id string) (Notifier, error)

	// Update performs an update to the existing notifier. A non-nil error is
	// returned to indicate operation failure.
	Update(ctx context.Context, n Notifier) error

	// Remove removes the notifiers having the provided identifiers
	Remove(ctx context.Context, ids ...string) error
}
