// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package smtp

import (
	"fmt"

	notifiers "github.com/MainfluxLabs/mainflux/consumers/notifiers"
	"github.com/MainfluxLabs/mainflux/internal/email"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

const (
	footer          = "Sent by Mainflux SMTP Notification"
	contentTemplate = "A publisher with an id %s sent the message over %s with the following values \n %s"
)

var _ notifiers.Notifier = (*notifier)(nil)

type notifier struct {
	agent *email.Agent
}

// New instantiates SMTP message notifier.
func New(agent *email.Agent) notifiers.Notifier {
	return &notifier{agent: agent}
}

func (n *notifier) Notify(from string, to []string, msg protomfx.Message) error {
	subject := fmt.Sprintf(`Mainflux notification: Thing %s and subtopic %s`, msg.Publisher, msg.Subtopic)
	values := string(msg.Payload)
	content := fmt.Sprintf(contentTemplate, msg.Publisher, msg.Protocol, values)

	return n.agent.Send(to, from, subject, "", content, footer)
}
