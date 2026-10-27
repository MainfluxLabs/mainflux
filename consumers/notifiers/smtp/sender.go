// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package smtp

import (
	"fmt"
	"strings"

	"github.com/MainfluxLabs/mainflux/consumers/notifiers"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/email"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

const (
	messageTemplate = "notification"
	alarmTemplate   = "alarm"
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

func (n *sender) Send(to []string, notification protomfx.Notification) error {
	if notification.Alarm != nil {
		return n.sendAlarm(to, notification)
	}

	return n.sendMessage(to, notification)
}

func (n *sender) sendMessage(to []string, notification protomfx.Notification) error {
	subject := fmt.Sprintf("New IoT message from Thing %s", notification.ThingId)
	if notification.Subtopic != "" {
		subject = fmt.Sprintf("New IoT message on %s from Thing %s", notification.Subtopic, notification.ThingId)
	}

	templateData := map[string]any{
		"ThingID":  notification.ThingId,
		"Subtopic": notification.Subtopic,
		"Protocol": notification.Protocol,
		"Time":     formatTime(notification.Created),
	}

	if fields, ok := payloadFields(notification.Payload); ok {
		templateData["Fields"] = fields
	} else {
		templateData["MessageContent"] = formatPayload(notification.Payload)
	}

	return n.agent.Send(to, n.from, subject, messageTemplate, templateData)
}

func (n *sender) sendAlarm(to []string, notification protomfx.Notification) error {
	alarm := notification.Alarm
	label := levelLabels[alarm.Level]
	upperLabel := strings.ToUpper(label)

	subject := fmt.Sprintf("[%s] Alarm from Thing %s", upperLabel, notification.ThingId)
	if notification.Subtopic != "" {
		subject = fmt.Sprintf("[%s] Alarm on %s from Thing %s", upperLabel, notification.Subtopic, notification.ThingId)
	}

	templateData := map[string]any{
		"ThingID":    notification.ThingId,
		"Subtopic":   notification.Subtopic,
		"Protocol":   notification.Protocol,
		"Time":       formatTime(notification.Created),
		"LevelLabel": label,
		"LevelClass": strings.ToLower(label),
		"RuleID":     alarm.RuleId,
		"Conditions": formatConditions(alarm.RuleInfo),
	}

	return n.agent.Send(to, n.from, subject, alarmTemplate, templateData)
}

func (n *sender) ValidateContacts(contacts []string) error {
	for _, c := range contacts {
		if !email.IsEmail(c) {
			return apiutil.ErrInvalidContact
		}
	}

	return nil
}
