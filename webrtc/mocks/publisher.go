// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"sync"

	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

var _ messaging.CommandPublisher = (*Publisher)(nil)

// PublishedCommand is a command recorded by the publisher.
type PublishedCommand struct {
	Subject string
	Command protomfx.Command
}

// Publisher is a command publisher that records what was published.
type Publisher struct {
	mu       sync.Mutex
	commands []PublishedCommand
}

// NewPublisher returns a command publisher that records what was published.
func NewPublisher() *Publisher {
	return &Publisher{}
}

// PublishCommand records the command.
func (p *Publisher) PublishCommand(subject string, cmd protomfx.Command) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.commands = append(p.commands, PublishedCommand{Subject: subject, Command: cmd})

	return nil
}

// Commands returns a copy of the recorded commands.
func (p *Publisher) Commands() []PublishedCommand {
	p.mu.Lock()
	defer p.mu.Unlock()

	return append([]PublishedCommand(nil), p.commands...)
}
