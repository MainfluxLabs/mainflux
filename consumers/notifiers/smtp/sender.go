// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package smtp

import (
	"fmt"
	"strings"

	"github.com/MainfluxLabs/mainflux/consumers/notifiers"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/email"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

const (
	messageTemplate = "notification"
	alarmTemplate   = "alarm"

	orgIDPlaceholder   = "{org_id}"
	groupIDPlaceholder = "{group_id}"
	thingIDPlaceholder = "{thing_id}"
	ruleIDPlaceholder  = "{rule_id}"
)

var _ notifiers.Sender = (*sender)(nil)

type sender struct {
	agent       *email.Agent
	from        string
	linksConfig LinksConfig
}

// LinksConfig configures the UI links in emails.
type LinksConfig struct {
	Host      string
	ThingPath string
	RulePath  string
}

// links builds the UI links of the group that a notification belongs to.
type links struct {
	config LinksConfig
	group  domain.Group
}

// notificationEmail holds the parts of the email sent for a notification.
type notificationEmail struct {
	subject      string
	templateName string
	templateData map[string]any
}

// New instantiates SMTP message sender.
func New(agent *email.Agent, from string, linksConfig LinksConfig) notifiers.Sender {
	return &sender{agent: agent, from: from, linksConfig: linksConfig}
}

func (n *sender) Send(to []string, notification protomfx.Notification, group domain.Group) error {
	groupLinks := links{
		config: n.linksConfig,
		group:  group,
	}

	content, err := buildEmail(notification, groupLinks)
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

func (l links) thingURL(thingID string) string {
	return l.url(l.config.ThingPath, thingIDPlaceholder, thingID)
}

func (l links) ruleURL(ruleID string) string {
	return l.url(l.config.RulePath, ruleIDPlaceholder, ruleID)
}

// url fills in the placeholders of the path
func (l links) url(path, idPlaceholder, id string) string {
	if path == "" || l.group.OrgID == "" || id == "" {
		return ""
	}

	replacer := strings.NewReplacer(
		orgIDPlaceholder, l.group.OrgID,
		groupIDPlaceholder, l.group.ID,
		idPlaceholder, id,
	)

	return l.config.Host + replacer.Replace(path)
}

// buildEmail returns the email for the alarm or the message that triggered the notification.
func buildEmail(notification protomfx.Notification, groupLinks links) (notificationEmail, error) {
	if alarm := notification.GetAlarm(); alarm != nil {
		return alarmEmail(alarm, groupLinks), nil
	}

	if notification.GetMessage() != nil {
		return messageEmail(notification, groupLinks)
	}

	return notificationEmail{}, notifiers.ErrMissingSource
}

func messageEmail(notification protomfx.Notification, groupLinks links) (notificationEmail, error) {
	message := notification.GetMessage()

	subject := fmt.Sprintf("New IoT message from Thing %s", message.ThingId)
	if message.Subtopic != "" {
		subject = fmt.Sprintf(
			"New IoT message on %s from Thing %s",
			message.Subtopic,
			message.ThingId,
		)
	}

	templateData := map[string]any{
		"ThingID":    message.ThingId,
		"ThingURL":   groupLinks.thingURL(message.ThingId),
		"Subtopic":   message.Subtopic,
		"Protocol":   message.Protocol,
		"Time":       formatTime(message.Created),
		"RuleID":     notification.RuleId,
		"RuleURL":    groupLinks.ruleURL(notification.RuleId),
		"Conditions": notifiers.FormatConditions(notification.RuleInfo),
	}

	if fields, ok := payloadFields(message); ok {
		templateData["Fields"] = fields
	} else {
		payload, err := messaging.ToPayload(*message)
		if err != nil {
			return notificationEmail{}, err
		}

		templateData["MessageContent"] = formatPayload(payload)
	}

	return notificationEmail{
		subject:      subject,
		templateName: messageTemplate,
		templateData: templateData,
	}, nil
}

func alarmEmail(alarm *protomfx.Alarm, groupLinks links) notificationEmail {
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
		"ThingURL":   groupLinks.thingURL(alarm.ThingId),
		"Subtopic":   alarm.Subtopic,
		"Protocol":   alarm.Protocol,
		"Time":       formatTime(alarm.Created),
		"LevelLabel": label,
		"LevelClass": strings.ToLower(label),
		"RuleID":     alarm.RuleId,
		"RuleURL":    groupLinks.ruleURL(alarm.RuleId),
		"Conditions": notifiers.FormatConditions(alarm.RuleInfo),
	}

	return notificationEmail{
		subject:      subject,
		templateName: alarmTemplate,
		templateData: templateData,
	}
}
