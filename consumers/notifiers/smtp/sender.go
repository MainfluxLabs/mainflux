// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package smtp

import (
	"fmt"

	"github.com/MainfluxLabs/mainflux/consumers/notifiers"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/email"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

var _ notifiers.Sender = (*sender)(nil)

type sender struct {
	agent *email.Agent
	from  string
}

// New instantiates SMTP message sender.
func New(agent *email.Agent, from string) notifiers.Sender {
	return &sender{agent: agent, from: from}
}

func (n *sender) Send(to []string, msg protomfx.Message) error {
	subject := fmt.Sprintf("New IoT message from Thing %s", msg.Publisher)
	if msg.Subtopic != "" {
		subject = fmt.Sprintf("New IoT message on %s from Thing %s", msg.Subtopic, msg.Publisher)
	}
	values := string(msg.Payload)

	templateData := map[string]any{
		"PublisherID":    msg.Publisher,
		"Protocol":       msg.Protocol,
		"MessageContent": values,
	}

	return n.agent.Send(to, n.from, subject, "notification", templateData)
}

func (n *sender) ValidateContacts(contacts []string) error {
	for _, c := range contacts {
		if !email.IsEmail(c) {
			return apiutil.ErrInvalidContact
		}
	}

	return nil
}
