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

// notificationEmail holds the parts of the email sent for a notification.
type notificationEmail struct {
	subject      string
	templateName string
	templateData map[string]any
}

// New instantiates SMTP message sender.
func New(agent *email.Agent, from string) notifiers.Sender {
	return &sender{agent: agent, from: from}
}

func (n *sender) Send(to []string, notification protomfx.Notification) error {
	content, err := buildEmail(notification)
	if err != nil {
		return err
	}

	return n.agent.Send(to, n.from, content.subject, content.templateName, content.templateData)
}

func (n *sender) ValidateContacts(contacts []string) error {
	for _, c := range contacts {
		if !email.IsEmail(c) {
			return apiutil.ErrInvalidContact
		}
	}

	return nil
}

// buildEmail returns the email for the alarm or the message that triggered the notification.
func buildEmail(notification protomfx.Notification) (notificationEmail, error) {
	if alarm := notification.GetAlarm(); alarm != nil {
		return alarmEmail(alarm), nil
	}

	if message := notification.GetMessage(); message != nil {
		return messageEmail(message), nil
	}

	return notificationEmail{}, notifiers.ErrMissingSource
}

func messageEmail(message *protomfx.Message) notificationEmail {
	subject := fmt.Sprintf("New IoT message from Thing %s", message.ThingId)
	if message.Subtopic != "" {
		subject = fmt.Sprintf(
			"New IoT message on %s from Thing %s",
			message.Subtopic,
			message.ThingId,
		)
	}

	templateData := map[string]any{
		"ThingID":  message.ThingId,
		"Subtopic": message.Subtopic,
		"Protocol": message.Protocol,
		"Time":     formatTime(message.Created),
	}

	if fields, ok := payloadFields(message.Payload, message.ContentType); ok {
		templateData["Fields"] = fields
	} else {
		templateData["MessageContent"] = formatPayload(message.Payload)
	}

	return notificationEmail{
		subject:      subject,
		templateName: messageTemplate,
		templateData: templateData,
	}
}

func alarmEmail(alarm *protomfx.Alarm) notificationEmail {
	label := notifiers.AlarmLevelLabel(alarm.Level)
	upperLabel := strings.ToUpper(label)

	subject := fmt.Sprintf("[%s] Alarm from Thing %s", upperLabel, alarm.ThingId)
	if alarm.Subtopic != "" {
		subject = fmt.Sprintf(
			"[%s] Alarm on %s from Thing %s",
			upperLabel,
			alarm.Subtopic,
			alarm.ThingId,
		)
	}

	templateData := map[string]any{
		"ThingID":    alarm.ThingId,
		"Subtopic":   alarm.Subtopic,
		"Protocol":   alarm.Protocol,
		"Time":       formatTime(alarm.Created),
		"LevelLabel": label,
		"LevelClass": strings.ToLower(label),
		"RuleID":     alarm.RuleId,
		"Conditions": notifiers.FormatConditions(alarm.RuleInfo),
	}

	return notificationEmail{
		subject:      subject,
		templateName: alarmTemplate,
		templateData: templateData,
	}
}
