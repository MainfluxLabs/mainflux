// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	notifiers "github.com/MainfluxLabs/mainflux/consumers/notifiers"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
)

var _ notifiers.Notifier = (*notifier)(nil)

const invalidSender = "invalid@example.com"

type notifier struct{}

// NewNotifier returns a new Notifier mock.
func NewNotifier() notifiers.Notifier {
	return notifier{}
}

func (n notifier) Notify(from string, to []string, msg messaging.Message) error {
	if len(to) < 1 {
		return notifiers.ErrNotify
	}

	for _, t := range to {
		if t == invalidSender {
			return notifiers.ErrNotify
		}
	}

	return nil
}
