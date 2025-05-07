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

const (
	footer          = "Sent by Mainflux SMTP Notification"
	contentTemplate = "A publisher with an id %s sent the message over %s with the following values \n %s"
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
	subject := fmt.Sprintf(`Mainflux notification: Thing %s and subtopic %s`, msg.Publisher, msg.Subtopic)
	values := string(msg.Payload)
	content := fmt.Sprintf(contentTemplate, msg.Publisher, msg.Protocol, values)

	return n.agent.Send(to, n.from, subject, "", content, footer)
}

func (n *sender) ValidateContacts(contacts []string) error {
	for _, c := range contacts {
		if !email.IsEmail(c) {
			return apiutil.ErrInvalidContact
		}
	}

	return nil
}
