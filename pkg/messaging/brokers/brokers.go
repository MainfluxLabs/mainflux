// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package brokers exposes broker-agnostic factories that select a concrete
// message broker implementation at build time via build tags.
package brokers

import "github.com/MainfluxLabs/mainflux/pkg/messaging"

// Publisher is the broker-agnostic publisher contract.
// It composes the base messaging.Publisher with command publishing.
type Publisher interface {
	messaging.Publisher
	messaging.CommandPublisher
}

// PubSub is the broker-agnostic pub/sub contract.
// It composes the base messaging.PubSub with command publishing.
type PubSub interface {
	messaging.PubSub
	messaging.CommandPublisher
}
