package rules

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

const (
	InputTypeMessage = "message"
	InputTypeAlarm   = "alarm"
)

type InputConfig map[string]any

func (c InputConfig) Subtopic() string {
	s, _ := c["subtopic"].(string)
	return s
}

type Input struct {
	Type     string      `json:"type"`
	ThingIDs []string    `json:"thing_ids"`
	Config   InputConfig `json:"config,omitempty"`
}

type Rule struct {
	ID          string
	GroupID     string
	Name        string
	Description string
	Input       Input
	Conditions  []Condition
	Operator    string
	Actions     []Action
}

type Condition = domain.Condition

type Action struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Level int32  `json:"level,omitempty"`
}

type RulesPage struct {
	Total uint64
	Rules []Rule
}

const (
	ActionTypeSMTP    = "smtp"
	ActionTypeSMPP    = "smpp"
	ActionTypeAlarm   = "alarm"
	ActionTypeWebhook = "webhook"

	OperatorAND = "AND"
	OperatorOR  = "OR"

	ComparatorEQ  = "=="
	ComparatorGTE = ">="
	ComparatorLTE = "<="
	ComparatorGT  = ">"
	ComparatorLT  = "<"

	// ConditionTypeThreshold compares a payload field against a numeric threshold.
	ConditionTypeThreshold = "threshold"
	// ConditionTypeScript runs a Lua script and uses its boolean return value.
	ConditionTypeScript = "script"
)

func (rs *rulesService) runRule(ctx context.Context, msg *protomfx.Message, parsedPayload any, rule Rule) error {
	triggered, err := rs.evaluateMessagePayload(ctx, msg, parsedPayload, rule)
	if err != nil {
		return err
	}
	if !triggered {
		return nil
	}

	for _, action := range rule.Actions {
		switch action.Type {
		case ActionTypeAlarm:
			ruleInfo, err := json.Marshal(domain.RuleInfo{Conditions: rule.Conditions, Operator: rule.Operator})
			if err != nil {
				return err
			}
			if err := rs.pub.PublishAlarm(fmt.Sprintf("%s.%s", subjectAlarms, domain.AlarmOriginRule), protomfx.Alarm{
				ThingId:  msg.ThingId,
				Subtopic: msg.Subtopic,
				Protocol: msg.Protocol,
				Created:  msg.Created,
				Level:    action.Level,
				RuleId:   rule.ID,
				RuleInfo: ruleInfo,
			}); err != nil {
				return err
			}
		case ActionTypeSMTP, ActionTypeSMPP:
			notification := protomfx.Notification{
				ThingId:  msg.ThingId,
				Subtopic: msg.Subtopic,
				Protocol: msg.Protocol,
				Payload:  msg.Payload,
				Created:  msg.Created,
			}
			if err := rs.pub.PublishNotification(fmt.Sprintf("%s.%s", action.Type, action.ID), notification); err != nil {
				return err
			}
		case ActionTypeWebhook:
			webhook := protomfx.Webhook{
				ThingId: msg.ThingId,
				Payload: msg.Payload,
				Created: msg.Created,
			}
			if err := rs.pub.PublishWebhook(subjectWebhooks, webhook); err != nil {
				return err
			}
		}
	}

	return nil
}

// evaluateAlarmPayload evaluates an alarm-input rule's conditions against payload
func evaluateAlarmPayload(payload any, conditions []Condition, operator string, contentType string) (bool, error) {
	switch data := payload.(type) {
	case []any:
		for _, item := range data {
			obj, ok := item.(map[string]any)
			if !ok {
				continue
			}
			triggered, err := evaluateAlarmConditions(obj, conditions, operator, contentType)
			if err != nil {
				return false, err
			}
			if triggered {
				return true, nil
			}
		}
		return false, nil
	case map[string]any:
		return evaluateAlarmConditions(data, conditions, operator, contentType)
	default:
		return false, errors.ErrInvalidPayload
	}
}

// evaluateAlarmConditions evaluates every condition of an alarm-input rule against a single
// payload object and reduces the results under the rule's operator.
func evaluateAlarmConditions(payloadMap map[string]any, conditions []Condition, operator, contentType string) (bool, error) {
	results := make([]bool, len(conditions))

	for i, condition := range conditions {
		r, err := evaluateThreshold(payloadMap, condition, contentType)
		if err != nil {
			return false, err
		}
		results[i] = r
	}

	return evaluateOperator(results, operator), nil
}

// evaluateMessagePayload evaluates a message-input rule's conditions against parsedPayload,
// which may run script conditions.
func (rs *rulesService) evaluateMessagePayload(ctx context.Context, msg *protomfx.Message, parsedPayload any, rule Rule) (bool, error) {
	switch data := parsedPayload.(type) {
	case []any:
		for _, item := range data {
			obj, ok := item.(map[string]any)
			if !ok {
				continue
			}
			triggered, err := rs.evaluateMessageConditions(ctx, msg, obj, rule)
			if err != nil {
				return false, err
			}
			if triggered {
				return true, nil
			}
		}
		return false, nil
	case map[string]any:
		return rs.evaluateMessageConditions(ctx, msg, data, rule)
	default:
		return false, errors.ErrInvalidPayload
	}
}

// evaluateMessageConditions evaluates every condition of a message-input rule against a
// single payload object, running script conditions along the way, and reduces the results
// under the rule's operator.
func (rs *rulesService) evaluateMessageConditions(ctx context.Context, msg *protomfx.Message, payload map[string]any, rule Rule) (bool, error) {
	results := make([]bool, len(rule.Conditions))

	for i, condition := range rule.Conditions {
		if condition.Type == ConditionTypeScript {
			results[i] = rs.runScriptCondition(ctx, msg, payload, rule.ID, condition.ScriptID)
			continue
		}

		r, err := evaluateThreshold(payload, condition, msg.ContentType)
		if err != nil {
			return false, err
		}
		results[i] = r
	}

	return evaluateOperator(results, rule.Operator), nil
}

// evaluateThreshold evaluates a single threshold condition against payload. A missing or
// non-numeric field is not met (false, nil); a field present but unparsable as a number
// is an error.
func evaluateThreshold(payload map[string]any, condition Condition, contentType string) (bool, error) {
	value := findPayloadParam(payload, condition.Field, contentType)
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
	case int:
		payloadValue = float64(v)
	case int32:
		payloadValue = float64(v)
	case int64:
		payloadValue = float64(v)
	case uint:
		payloadValue = float64(v)
	case uint64:
		payloadValue = float64(v)
	default:
		return false, nil
	}

	return isConditionMet(condition.Comparator, payloadValue, *condition.Threshold), nil
}

// evaluateOperator combines per-condition results under the rule's operator (AND if unset).
func evaluateOperator(results []bool, operator string) bool {
	if operator == OperatorOR {
		for _, r := range results {
			if r {
				return true
			}
		}
		return false
	}

	for _, r := range results {
		if !r {
			return false
		}
	}
	return true
}

func isConditionMet(comparator string, val1, val2 float64) bool {
	switch comparator {
	case ComparatorEQ:
		return val1 == val2
	case ComparatorGTE:
		return val1 >= val2
	case ComparatorLTE:
		return val1 <= val2
	case ComparatorGT:
		return val1 > val2
	case ComparatorLT:
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

	for _, key := range parts[:len(parts)-1] {
		nested, ok := current[key].(map[string]any)
		if !ok {
			return nil
		}
		current = nested
	}

	val, ok := current[parts[len(parts)-1]]
	if !ok {
		return nil
	}
	return val
}

// RuleOrderFields maps API-facing order keys to SQL column expressions for the rules table.
var RuleOrderFields = map[string]string{
	"id":   "id",
	"name": "LOWER(name)",
}

// ScriptRunOrderFields maps API-facing order keys to SQL column expressions for the lua_script_runs table.
var ScriptRunOrderFields = map[string]string{
	"id":          "id",
	"started_at":  "started_at",
	"finished_at": "finished_at",
	"status":      "status",
}
