package rules

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/MainfluxLabs/mainflux/consumers/alarms"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

type Rule struct {
	ID          string
	GroupID     string
	Name        string
	Description string
	Conditions  []Condition
	Operator    string
	Actions     []Action
}

type Condition struct {
	Field      string   `json:"field"`
	Comparator string   `json:"comparator"`
	Threshold  *float64 `json:"threshold"`
}

type Action struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type RulesPage struct {
	Total uint64
	Rules []Rule
}

const (
	ActionTypeSMTP  = "smtp"
	ActionTypeSMPP  = "smpp"
	ActionTypeAlarm = "alarm"

	OperatorAND = "AND"
	OperatorOR  = "OR"
)

var errInvalidObject = errors.New("invalid JSON object")

func (rs *rulesService) processRule(msg *protomfx.Message, parsedPayload any, rule Rule) error {
	triggered, payloads, err := processPayload(parsedPayload, rule.Conditions, rule.Operator, msg.ContentType)
	if err != nil {
		return err
	}

	if !triggered {
		return nil
	}

	for _, action := range rule.Actions {
		for _, payload := range payloads {
			newMsg := msg
			newMsg.Payload = payload

			switch action.Type {
			case ActionTypeSMTP:
				newMsg.Subject = fmt.Sprintf("%s.%s", subjectSMTP, action.ID)
			case ActionTypeSMPP:
				newMsg.Subject = fmt.Sprintf("%s.%s", subjectSMPP, action.ID)
			case ActionTypeAlarm:
				newMsg.Subject = fmt.Sprintf("%s.%s.%s", subjectAlarms, alarms.AlarmOriginRule, rule.ID)
			default:
				continue
			}

			if err := rs.pubsub.Publish(*newMsg); err != nil {
				return err
			}
		}
	}

	return nil
}

func processPayload(payload any, conditions []Condition, operator string, contentType string) (bool, [][]byte, error) {
	switch data := payload.(type) {
	case []any:
		var triggerPayloads [][]byte
		for _, item := range data {
			obj, ok := item.(map[string]any)
			if !ok {
				continue
			}

			triggered, err := checkConditionsMet(obj, conditions, operator, contentType)
			if err != nil {
				return false, nil, err
			}

			if triggered {
				extractedPayload, err := json.Marshal(obj)
				if err != nil {
					return false, nil, err
				}
				triggerPayloads = append(triggerPayloads, extractedPayload)
			}
		}

		return len(triggerPayloads) > 0, triggerPayloads, nil
	case map[string]any:
		triggered, err := checkConditionsMet(data, conditions, operator, contentType)
		if err != nil {
			return false, nil, err
		}

		if triggered {
			extractedPayload, err := json.Marshal(data)
			if err != nil {
				return false, nil, err
			}
			return true, [][]byte{extractedPayload}, nil
		}

		return false, nil, nil
	default:
		return false, nil, errInvalidObject
	}
}

func checkConditionsMet(payloadMap map[string]any, conditions []Condition, operator, contentType string) (bool, error) {
	results := make([]bool, len(conditions))

	for i, condition := range conditions {
		value := findPayloadParam(payloadMap, condition.Field, contentType)
		if value == nil {
			results[i] = false
			continue
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
		case int:
			payloadValue = float64(v)
		case int64:
			payloadValue = float64(v)
		case uint:
			payloadValue = float64(v)
		case uint64:
			payloadValue = float64(v)
		default:
			results[i] = false
			continue
		}

		results[i] = isConditionMet(condition.Comparator, payloadValue, *condition.Threshold)
	}

	if operator == OperatorOR {
		for _, r := range results {
			if r {
				return true, nil
			}
		}
		return false, nil
	}

	for _, r := range results {
		if !r {
			return false, nil
		}
	}
	return true, nil
}

func isConditionMet(comparator string, val1, val2 float64) bool {
	switch comparator {
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

func findPayloadParam(payload map[string]any, param string, contentType string) any {
	switch contentType {
	case messaging.SenMLContentType:
		if name, ok := payload["name"].(string); ok && name == param {
			if value, exists := payload["value"]; exists {
				return value
			}
		}
		return nil
	case messaging.JSONContentType:
		return findParam(payload, param)
	default:
		return nil
	}
}

func findParam(payload map[string]any, param string) any {
	if param == "" {
		return nil
	}

	parts := strings.Split(param, ".")
	current := payload

	for i, key := range parts {
		val, ok := current[key]
		if !ok {
			return nil
		}

		if i < len(parts)-1 {
			nested, ok := val.(map[string]any)
			if !ok {
				return nil
			}
			current = nested
		} else {
			return val
		}
	}
	return nil
}
