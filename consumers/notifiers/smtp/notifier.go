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

var _ notifiers.Notifier = (*notifier)(nil)

type notifier struct {
	agent *email.Agent
	from  string
}

// New instantiates SMTP message notifier.
func New(agent *email.Agent, from string) notifiers.Notifier {
	return &notifier{agent: agent, from: from}
}

func (n *notifier) Notify(to []string, msg protomfx.Message) error {
	subject := fmt.Sprintf(`Mainflux notification: Thing %s and subtopic %s`, msg.Publisher, msg.Subtopic)
	values := string(msg.Payload)
	content := fmt.Sprintf(contentTemplate, msg.Publisher, msg.Protocol, values)

	return n.agent.Send(to, n.from, subject, "", content, footer)
}

func (n *notifier) ValidateContacts(contacts []string) error {
	for _, c := range contacts {
		if !email.IsEmail(c) {
			return apiutil.ErrInvalidContact
		}
	}

	return nil
}
