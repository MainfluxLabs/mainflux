// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"sync"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

// Publisher covers the methods this mock's callers (http/mqtt/modbus/downlinks,
// pkg/sdk tests) use.
type Publisher interface {
	messaging.CommandPublisher
	messaging.MessageDispatcher
	Publish(subject string, msg protomfx.Message) error
}

// PublishedMessage is a message recorded by a RecordingPublisher.
type PublishedMessage struct {
	Subject string
	Message protomfx.Message
}

// PublishedCommand is a command recorded by a RecordingPublisher.
type PublishedCommand struct {
	Subject string
	Command protomfx.Command
}

// RecordingPublisher is a Publisher that records what was published.
type RecordingPublisher interface {
	Publisher
	Published() []PublishedMessage
	Commands() []PublishedCommand
	Dispatched() []protomfx.Message
}

type mockPublisher struct{}

// NewPublisher returns mock message publisher.
func NewPublisher() Publisher {
	return mockPublisher{}
}

func (pub mockPublisher) Publish(_ string, _ protomfx.Message) error {
	return nil
}

func (pub mockPublisher) PublishCommand(_ string, cmd protomfx.Command) error {
	return nil
}

func (pub mockPublisher) Dispatch(_ protomfx.Message, _ *domain.ProfileConfig) error {
	return nil
}

var _ RecordingPublisher = (*recordingPublisher)(nil)

type recordingPublisher struct {
	mu         sync.Mutex
	published  []PublishedMessage
	commands   []PublishedCommand
	dispatched []protomfx.Message
}

// NewRecordingPublisher returns a mock publisher that records every call.
func NewRecordingPublisher() RecordingPublisher {
	return &recordingPublisher{}
}

func (pub *recordingPublisher) Publish(subject string, msg protomfx.Message) error {
	pub.mu.Lock()
	defer pub.mu.Unlock()

	pub.published = append(pub.published, PublishedMessage{Subject: subject, Message: msg})
	return nil
}

func (pub *recordingPublisher) PublishCommand(subject string, cmd protomfx.Command) error {
	pub.mu.Lock()
	defer pub.mu.Unlock()

	pub.commands = append(pub.commands, PublishedCommand{Subject: subject, Command: cmd})
	return nil
}

func (pub *recordingPublisher) Dispatch(msg protomfx.Message, _ *domain.ProfileConfig) error {
	pub.mu.Lock()
	defer pub.mu.Unlock()

	pub.dispatched = append(pub.dispatched, msg)
	return nil
}

func (pub *recordingPublisher) Published() []PublishedMessage {
	pub.mu.Lock()
	defer pub.mu.Unlock()

	return append([]PublishedMessage(nil), pub.published...)
}

func (pub *recordingPublisher) Commands() []PublishedCommand {
	pub.mu.Lock()
	defer pub.mu.Unlock()

	return append([]PublishedCommand(nil), pub.commands...)
}

func (pub *recordingPublisher) Dispatched() []protomfx.Message {
	pub.mu.Lock()
	defer pub.mu.Unlock()

	return append([]protomfx.Message(nil), pub.dispatched...)
}
