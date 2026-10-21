// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"sync"

	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

var _ messaging.CommandPublisher = (*CommandPublisher)(nil)

// CommandPublisher records published commands by subject.
type CommandPublisher struct {
	mu       sync.Mutex
	commands map[string][]protomfx.Command
}

// NewCommandPublisher returns a command publisher that records what it publishes.
func NewCommandPublisher() *CommandPublisher {
	return &CommandPublisher{commands: make(map[string][]protomfx.Command)}
}

func (pub *CommandPublisher) PublishCommand(subject string, cmd protomfx.Command) error {
	pub.mu.Lock()
	defer pub.mu.Unlock()

	pub.commands[subject] = append(pub.commands[subject], cmd)

	return nil
}

// Commands returns the commands published to the given subject.
func (pub *CommandPublisher) Commands(subject string) []protomfx.Command {
	pub.mu.Lock()
	defer pub.mu.Unlock()

	return pub.commands[subject]
}
