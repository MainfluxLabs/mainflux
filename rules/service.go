// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package rules

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
)

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
// All methods that accept a token parameter use it to identify and authorize
// the user performing the operation.
type Service interface {
	// CreateRules creates rules.
	CreateRules(ctx context.Context, token, groupID string, rules ...Rule) ([]Rule, error)

	// ListRulesByThing retrieves a paginated list of rules by thing.
	ListRulesByThing(ctx context.Context, token, thingID string, pm apiutil.PageMetadata) (RulesPage, error)

	// ListRulesByGroup retrieves a paginated list of rules by group.
	ListRulesByGroup(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (RulesPage, error)

	// ListThingIDsByRule retrieves a list of thing IDs attached to the given rule ID.
	ListThingIDsByRule(ctx context.Context, token, ruleID string) ([]string, error)

	// ViewRule retrieves a specific rule by its ID.
	ViewRule(ctx context.Context, token, id string) (Rule, error)

	// UpdateRule updates the rule identified by the provided ID.
	UpdateRule(ctx context.Context, token string, rule Rule) error

	// RemoveRules removes the rules identified with the provided IDs.
	RemoveRules(ctx context.Context, token string, ids ...string) error

	// RemoveRulesByGroup removes the rules identified with the provided IDs.
	RemoveRulesByGroup(ctx context.Context, groupID string) error

	// AssignRules assigns rules to a specific thing.
	AssignRules(ctx context.Context, token, thingID string, ruleIDs ...string) error

	// UnassignRules removes rule assignments from a specific thing.
	UnassignRules(ctx context.Context, token, thingID string, ruleIDs ...string) error

	// UnassignRulesByThing removes all rule assignments from a specific thing.
	UnassignRulesByThing(ctx context.Context, thingID string) error

	// Publish publishes messages on a topic related to a certain rule action
	Publish(ctx context.Context, message protomfx.Message) error
}

const (
	ActionTypeSMTP  = "smtp"
	ActionTypeSMPP  = "smpp"
	ActionTypeAlarm = "alarm"

	OperatorAND = "AND"
	OperatorOR  = "OR"

	// subjectMessages represents subject used to publish all incoming messages.
	subjectMessages = "messages"
	// subjectWriters represents subject used to publish messages that should be persisted.
	subjectWriters = "writers"
	// subjectAlarms represents subject used to publish messages that trigger an alarm.
	subjectAlarms = "alarms"
	// subjectSMTP represents subject used to publish messages that trigger an SMTP notification.
	subjectSMTP = "smtp"
	// subjectSMPP represents subject used to publish messages that trigger an SMPP notification.
	subjectSMPP = "smpp"
)

var errInvalidObject = errors.New("invalid JSON object")

type rulesService struct {
	rules      RuleRepository
	things     protomfx.ThingsServiceClient
	publisher  messaging.Publisher
	idProvider uuid.IDProvider
	logger     logger.Logger
}

var _ Service = (*rulesService)(nil)

// New instantiates the rules service implementation.
func New(rules RuleRepository, things protomfx.ThingsServiceClient, publisher messaging.Publisher, idp uuid.IDProvider, logger logger.Logger) Service {
	return &rulesService{
		rules:      rules,
		things:     things,
		publisher:  publisher,
		idProvider: idp,
		logger:     logger,
	}
}

func (rs *rulesService) CreateRules(ctx context.Context, token, groupID string, rules ...Rule) ([]Rule, error) {
	_, err := rs.things.CanUserAccessGroup(ctx, &protomfx.UserAccessReq{Token: token, Id: groupID, Action: things.Editor})
	if err != nil {
		return []Rule{}, err
	}

	for i := range rules {
		rules[i].GroupID = groupID

		id, err := rs.idProvider.ID()
		if err != nil {
			return []Rule{}, err
		}
		rules[i].ID = id
	}

	return rs.rules.Save(ctx, rules...)
}

func (rs *rulesService) ListRulesByThing(ctx context.Context, token, thingID string, pm apiutil.PageMetadata) (RulesPage, error) {
	_, err := rs.things.CanUserAccessThing(ctx, &protomfx.UserAccessReq{Token: token, Id: thingID, Action: things.Viewer})
	if err != nil {
		return RulesPage{}, err
	}

	rules, err := rs.rules.RetrieveByThing(ctx, thingID, pm)
	if err != nil {
		return RulesPage{}, err
	}

	return rules, nil
}

func (rs *rulesService) ListRulesByGroup(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (RulesPage, error) {
	_, err := rs.things.CanUserAccessGroup(ctx, &protomfx.UserAccessReq{Token: token, Id: groupID, Action: things.Viewer})
	if err != nil {
		return RulesPage{}, err
	}

	rules, err := rs.rules.RetrieveByGroup(ctx, groupID, pm)
	if err != nil {
		return RulesPage{}, err
	}

	return rules, nil
}

func (rs *rulesService) ListThingIDsByRule(ctx context.Context, token, ruleID string) ([]string, error) {
	rule, err := rs.rules.RetrieveByID(ctx, ruleID)
	if err != nil {
		return []string{}, err
	}

	if _, err := rs.things.CanUserAccessGroup(ctx, &protomfx.UserAccessReq{Token: token, Id: rule.GroupID, Action: things.Viewer}); err != nil {
		return []string{}, err
	}

	return rs.rules.RetrieveThingIDsByRule(ctx, ruleID)
}

func (rs *rulesService) ViewRule(ctx context.Context, token, id string) (Rule, error) {
	rule, err := rs.rules.RetrieveByID(ctx, id)
	if err != nil {
		return Rule{}, err
	}

	if _, err := rs.things.CanUserAccessGroup(ctx, &protomfx.UserAccessReq{Token: token, Id: rule.GroupID, Action: things.Viewer}); err != nil {
		return Rule{}, err
	}

	return rule, nil
}

func (rs *rulesService) UpdateRule(ctx context.Context, token string, rule Rule) error {
	r, err := rs.rules.RetrieveByID(ctx, rule.ID)
	if err != nil {
		return err
	}

	req := &protomfx.UserAccessReq{Token: token, Id: r.GroupID, Action: things.Editor}
	if _, err := rs.things.CanUserAccessGroup(ctx, req); err != nil {
		return err
	}

	return rs.rules.Update(ctx, rule)
}

func (rs *rulesService) RemoveRules(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		rule, err := rs.rules.RetrieveByID(ctx, id)
		if err != nil {
			return err
		}

		if _, err := rs.things.CanUserAccessGroup(ctx, &protomfx.UserAccessReq{Token: token, Id: rule.GroupID, Action: things.Editor}); err != nil {
			return err
		}
	}

	return rs.rules.Remove(ctx, ids...)
}

func (rs *rulesService) RemoveRulesByGroup(ctx context.Context, groupID string) error {
	return rs.rules.RemoveByGroup(ctx, groupID)
}

func (rs *rulesService) AssignRules(ctx context.Context, token, thingID string, ruleIDs ...string) error {
	if _, err := rs.things.CanUserAccessThing(ctx, &protomfx.UserAccessReq{Token: token, Id: thingID, Action: things.Editor}); err != nil {
		return err
	}

	grID, err := rs.things.GetGroupIDByThingID(ctx, &protomfx.ThingID{Value: thingID})
	if err != nil {
		return err
	}

	for _, id := range ruleIDs {
		rule, err := rs.rules.RetrieveByID(ctx, id)
		if err != nil {
			return err
		}

		if rule.GroupID != grID.GetValue() {
			return errors.ErrAuthorization
		}
	}

	if err := rs.rules.Assign(ctx, thingID, ruleIDs...); err != nil {
		return err
	}

	return nil
}

func (rs *rulesService) UnassignRules(ctx context.Context, token, thingID string, ruleIDs ...string) error {
	if _, err := rs.things.CanUserAccessThing(ctx, &protomfx.UserAccessReq{Token: token, Id: thingID, Action: things.Editor}); err != nil {
		return err
	}

	grID, err := rs.things.GetGroupIDByThingID(ctx, &protomfx.ThingID{Value: thingID})
	if err != nil {
		return err
	}

	for _, id := range ruleIDs {
		rule, err := rs.rules.RetrieveByID(ctx, id)
		if err != nil {
			return err
		}

		if rule.GroupID != grID.GetValue() {
			return errors.ErrAuthorization
		}
	}

	if err := rs.rules.Unassign(ctx, thingID, ruleIDs...); err != nil {
		return err
	}

	return nil
}

func (rs *rulesService) UnassignRulesByThing(ctx context.Context, thingID string) error {
	return rs.rules.UnassignByThing(ctx, thingID)
}

func (rs *rulesService) Publish(ctx context.Context, message protomfx.Message) error {
	rp, err := rs.rules.RetrieveByThing(ctx, message.GetPublisher(), apiutil.PageMetadata{})
	if err != nil {
		return err
	}

	// publish a message to default subjects independently of rule check
	go func(msg protomfx.Message) {
		if err := rs.publishToDefaultSubjects(msg); err != nil {
			rs.logger.Error(err.Error())
		}
	}(message)

	for _, rule := range rp.Rules {
		triggered, payloads, err := processPayload(message.Payload, rule.Conditions, rule.Operator, message.ContentType)
		if err != nil {
			return errors.Wrap(messaging.ErrPublishMessage, err)
		}
		if !triggered {
			continue
		}

		for _, action := range rule.Actions {
			for _, payload := range payloads {
				newMsg := message
				newMsg.Payload = payload

				switch action.Type {
				case ActionTypeSMTP:
					newMsg.Subject = fmt.Sprintf("%s.%s", subjectSMTP, action.ID)
				case ActionTypeSMPP:
					newMsg.Subject = fmt.Sprintf("%s.%s", subjectSMPP, action.ID)
				case ActionTypeAlarm:
					newMsg.Subject = fmt.Sprintf("%s.%s", subjectAlarms, rule.ID)
				default:
					continue
				}

				if err := rs.publisher.Publish(newMsg); err != nil {
					return errors.Wrap(messaging.ErrPublishMessage, err)
				}
			}
		}
	}

	return nil
}

func (rs *rulesService) publishToDefaultSubjects(msg protomfx.Message) error {
	subjects := []string{subjectMessages, subjectWriters}
	if msg.Subtopic != "" {
		subjects = append(subjects, fmt.Sprintf("%s.%s", subjectMessages, msg.Subtopic))
	}

	for _, sub := range subjects {
		m := msg
		m.Subject = sub
		if err := rs.publisher.Publish(m); err != nil {
			return errors.Wrap(messaging.ErrPublishMessage, err)
		}
	}
	return nil
}

func processPayload(payload []byte, conditions []Condition, operator string, contentType string) (bool, [][]byte, error) {
	var parsedData any
	if err := json.Unmarshal(payload, &parsedData); err != nil {
		return false, nil, err
	}

	switch data := parsedData.(type) {
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
