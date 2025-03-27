// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package nats

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/gogo/protobuf/proto"
	broker "github.com/nats-io/nats.go"
)

// A maximum number of reconnect attempts before NATS connection closes permanently.
// Value -1 represents an unlimited number of reconnect retries, i.e. the client
// will never give up on retrying to re-establish connection to NATS server.
const (
	maxReconnects  = -1
	messagesSuffix = "messages"
	subjectSMTP    = "smtp"
	subjectSMPP    = "smpp"
	subjectWebhook = "webhook"
	subjectAlarm   = "alarm"
)

var _ messaging.Publisher = (*publisher)(nil)

type publisher struct {
	conn *broker.Conn
}

// Publisher wraps messaging Publisher exposing
// Close() method for NATS connection.

// NewPublisher returns NATS message Publisher.
func NewPublisher(url string) (messaging.Publisher, error) {
	conn, err := broker.Connect(url, broker.MaxReconnects(maxReconnects))
	if err != nil {
		return nil, err
	}
	ret := &publisher{
		conn: conn,
	}
	return ret, nil
}

func (pub *publisher) Publish(msg protomfx.Message) (err error) {
	format, err := getFormat(msg.ProfileConfig.ContentType)
	if err != nil {
		return err
	}

	data, err := proto.Marshal(&msg)
	if err != nil {
		return err
	}

	subject := fmt.Sprintf("%s.%s", format, messagesSuffix)
	if msg.ProfileConfig.Write {
		if msg.Subtopic != "" {
			subject = fmt.Sprintf("%s.%s", subject, msg.Subtopic)
		}

		if err := pub.conn.Publish(subject, data); err != nil {
			return err
		}
	}

	if msg.ProfileConfig.Webhook {
		if err := pub.conn.Publish(subjectWebhook, data); err != nil {
			return err
		}
	}

	if msg.ProfileConfig.GetSmtpID() != "" {
		notification, err := createNotification(msg)
		if err != nil {
			return err
		}

		if err := pub.conn.Publish(subjectSMTP, notification); err != nil {
			return err
		}
	}

	if msg.ProfileConfig.GetSmppID() != "" {
		notification, err := createNotification(msg)
		if err != nil {
			return err
		}

		if err := pub.conn.Publish(subjectSMPP, notification); err != nil {
			return err
		}
	}

	if msg.ProfileConfig.GetRule() != nil {
		valid, err := isPayloadValidForRule(data, *msg.ProfileConfig.GetRule())
		if err != nil {
			return err
		}
		if !valid {
			alarm, err := createAlarm(msg)
			if err != nil {
				return err
			}

			if err := pub.conn.Publish(subjectAlarm, alarm); err != nil {
				return err
			}
		}
	}

	return nil
}

func (pub *publisher) Close() error {
	pub.conn.Close()
	return nil
}

func getFormat(ct string) (format string, err error) {
	switch ct {
	case messaging.JSONContentType:
		return messaging.JSONFormat, nil
	case messaging.SenMLContentType:
		return messaging.SenMLFormat, nil
	case messaging.CBORContentType:
		return messaging.CBORFormat, nil
	default:
		return messaging.SenMLFormat, nil
	}
}

func createNotification(msg protomfx.Message) ([]byte, error) {
	notification := protomfx.Notification{
		PublisherID: msg.GetPublisher(),
		Subtopic:    msg.GetSubtopic(),
		Payload:     msg.GetPayload(),
		Protocol:    msg.GetProtocol(),
		SmtpID:      msg.GetProfileConfig().GetSmtpID(),
		SmppID:      msg.GetProfileConfig().GetSmppID(),
	}

	data, err := proto.Marshal(&notification)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func createAlarm(msg protomfx.Message) ([]byte, error) {
	alarm := protomfx.Alarm{
		PublisherID: msg.GetPublisher(),
		Subtopic:    msg.GetSubtopic(),
		Protocol:    msg.GetProtocol(),
		Payload:     msg.GetPayload(),
		Rule:        msg.GetProfileConfig().GetRule(),
		Created:     msg.GetCreated(),
	}

	data, err := proto.Marshal(&alarm)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func isPayloadValidForRule(payload []byte, rule protomfx.Rule) (bool, error) {
	var (
		errInvalidValueType = errors.New("invalid value type")
		payloadMap          map[string]interface{}
	)

	if err := json.Unmarshal(payload, &payloadMap); err != nil {
		return false, err
	}

	value := messaging.FindParam(payloadMap, rule.GetField())
	if value == nil {
		return false, nil
	}

	var payloadValue float64
	switch v := value.(type) {
	case string:
		val, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return false, err
		}
		payloadValue = val
	case float64:
		payloadValue = v
	default:
		return false, errInvalidValueType
	}

	return isValidValue(rule.GetOperator(), payloadValue, float64(rule.GetValue())), nil
}

func isValidValue(operator string, val1, val2 float64) bool {
	switch operator {
	case "==":
		return val1 == val2
	case ">=":
		return val1 >= val2
	case "<=":
		return val1 <= val2
	case ">":
		return val1 > val2
	case "<":
		return val1 < val2
	default:
		return false
	}
}
