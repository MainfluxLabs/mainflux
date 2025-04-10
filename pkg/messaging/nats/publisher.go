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
	maxReconnects   = -1
	messagesSuffix  = "messages"
	subjectWebhook  = "webhook"
	subjectAlarm    = "alarm"
	actionTypeSMTP  = "smtp"
	actionTypeSMPP  = "smpp"
	actionTypeAlarm = "alarm"
)

var _ messaging.Publisher = (*publisher)(nil)
var (
	errInvalidActionID   = errors.New("invalid action id")
	errInvalidActionType = errors.New("invalid action type")
	errInvalidObject     = errors.New("invalid JSON object")
)

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
	data, err := proto.Marshal(&msg)
	if err != nil {
		return err
	}

	format := getFormat(msg.ContentType)
	subject := fmt.Sprintf("%s.%s", format, messagesSuffix)
	if msg.WriteEnabled {
		if msg.Subtopic != "" {
			subject = fmt.Sprintf("%s.%s", subject, msg.Subtopic)
		}

		if err := pub.conn.Publish(subject, data); err != nil {
			return err
		}
	}

	if err := pub.conn.Publish(subjectWebhook, data); err != nil {
		return err
	}

	if len(msg.Rules) > 0 {
		if err := pub.performRuleActions(&msg); err != nil {
			return err
		}
	}

	return nil
}

func (pub *publisher) performRuleActions(msg *protomfx.Message) error {
	for _, rule := range msg.Rules {
		if len(rule.Actions) == 0 {
			continue
		}

		isValid, payloads, err := processPayload(msg.Payload, *rule, msg.ContentType)
		if err != nil {
			return err
		}
		if isValid {
			continue
		}

		for _, action := range rule.Actions {
			for _, payload := range payloads {
				newMsg := *msg
				newMsg.Rules = []*protomfx.Rule{rule}
				newMsg.Payload = payload

				data, err := proto.Marshal(&newMsg)
				if err != nil {
					return err
				}

				switch action.Type {
				case actionTypeSMTP, actionTypeSMPP:
					if action.Id != "" {
						if err := pub.conn.Publish(action.Type, data); err != nil {
							return err
						}
					}
					return errInvalidActionID
				case actionTypeAlarm:
					if err := pub.conn.Publish(subjectAlarm, data); err != nil {
						return err
					}
				default:
					return errInvalidActionType
				}
			}
		}
	}

	return nil
}

func processPayload(payload []byte, rule protomfx.Rule, contentType string) (bool, [][]byte, error) {
	var parsedData interface{}
	if err := json.Unmarshal(payload, &parsedData); err != nil {
		return false, nil, err
	}

	switch data := parsedData.(type) {
	case []interface{}:
		var invalidPayloads [][]byte
		for _, item := range data {
			obj, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			isValid, err := validatePayload(obj, rule, contentType)
			if err != nil {
				return false, nil, err
			}

			if !isValid {
				extractedPayload, _ := json.Marshal(obj)
				invalidPayloads = append(invalidPayloads, extractedPayload)
			}
		}

		return len(invalidPayloads) == 0, invalidPayloads, nil
	case map[string]interface{}:
		isValid, err := validatePayload(data, rule, contentType)
		if err != nil {
			return false, nil, err
		}

		if !isValid {
			extractedPayload, _ := json.Marshal(data)
			return false, [][]byte{extractedPayload}, nil
		}

		return true, nil, nil
	default:
		return false, nil, errInvalidObject
	}
}

func validatePayload(payloadMap map[string]interface{}, rule protomfx.Rule, contentType string) (bool, error) {
	value := findPayloadParam(payloadMap, rule.Field, contentType)
	if value == nil {
		return true, nil
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
		return false, nil
	}

	return !isConditionMet(rule.Operator, payloadValue, rule.Threshold), nil
}

func isConditionMet(operator string, val1, val2 float64) bool {
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

func findPayloadParam(payload map[string]interface{}, param string, contentType string) interface{} {
	switch contentType {
	case messaging.SenMLContentType:
		if name, ok := payload["n"].(string); ok && name == param {
			if value, exists := payload["v"]; exists {
				return value
			}
		}
		return nil
	case messaging.JSONContentType:
		return messaging.FindParam(payload, param)
	default:
		return nil
	}
}

func getFormat(ct string) string {
	switch ct {
	case messaging.JSONContentType:
		return messaging.JSONFormat
	case messaging.SenMLContentType:
		return messaging.SenMLFormat
	case messaging.CBORContentType:
		return messaging.CBORFormat
	default:
		return messaging.SenMLFormat
	}
}

func (pub *publisher) Close() error {
	pub.conn.Close()
	return nil
}
