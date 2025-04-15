// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package rules

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
)

// Service specifies an API for managing rules.
type Service interface {
	// CreateRules creates rules.
	CreateRules(ctx context.Context, token string, rules ...Rule) ([]Rule, error)

	// ListRulesByProfile retrieves a paginated list of rules by profile.
	ListRulesByProfile(ctx context.Context, token, profileID string, pm apiutil.PageMetadata) (RulesPage, error)

	// ListRulesByGroup retrieves a paginated list of rules by group.
	ListRulesByGroup(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (RulesPage, error)

	// ViewRule retrieves a specific rule by its ID.
	ViewRule(ctx context.Context, token, id string) (Rule, error)

	// UpdateRule updates the rule identified by the provided ID.
	UpdateRule(ctx context.Context, token string, rule Rule) error

	// RemoveRules removes the rules identified with the provided IDs.
	RemoveRules(ctx context.Context, token string, ids ...string) error

	// Publish publishes messages on a topic related to a certain rule action
	Publish(ctx context.Context, message protomfx.Message) error
}

const (
	actionTypeSMTP  = "smtp"
	actionTypeSMPP  = "smpp"
	actionTypeAlarm = "alarm"
)

var (
	errInvalidActionID   = errors.New("invalid action id")
	errInvalidActionType = errors.New("invalid action type")
	errInvalidObject     = errors.New("invalid JSON object")
)

type rulesService struct {
	rules      RuleRepository
	things     protomfx.ThingsServiceClient
	publisher  messaging.Publisher
	idProvider uuid.IDProvider
}

var _ Service = (*rulesService)(nil)

// New instantiates the rules service implementation.
func New(rules RuleRepository, things protomfx.ThingsServiceClient, publisher messaging.Publisher, idp uuid.IDProvider) Service {
	return &rulesService{
		rules:      rules,
		things:     things,
		publisher:  publisher,
		idProvider: idp,
	}
}

func (rs *rulesService) CreateRules(ctx context.Context, token string, rules ...Rule) ([]Rule, error) {
	var rls []Rule
	for _, rule := range rules {
		r, err := rs.createRule(ctx, &rule, token)
		if err != nil {
			return []Rule{}, err
		}
		rls = append(rls, r)
	}

	return rls, nil
}

func (rs *rulesService) createRule(ctx context.Context, rule *Rule, token string) (Rule, error) {
	_, err := rs.things.CanUserAccessProfile(ctx, &protomfx.UserAccessReq{Token: token, Id: rule.ProfileID, Action: things.Editor})
	if err != nil {
		return Rule{}, err
	}

	//TODO: Replace by GetGroupIDByProfileID
	grID, err := rs.things.GetGroupIDByThingID(ctx, &protomfx.ThingID{Value: rule.ProfileID})
	if err != nil {
		return Rule{}, err
	}
	rule.GroupID = grID.GetValue()

	id, err := rs.idProvider.ID()
	if err != nil {
		return Rule{}, err
	}
	rule.ID = id

	rls, err := rs.rules.Save(ctx, *rule)
	if err != nil {
		return Rule{}, err
	}

	if len(rls) == 0 {
		return Rule{}, errors.ErrCreateEntity
	}

	return rls[0], nil
}

func (rs *rulesService) ListRulesByProfile(ctx context.Context, token, profileID string, pm apiutil.PageMetadata) (RulesPage, error) {
	_, err := rs.things.CanUserAccessProfile(ctx, &protomfx.UserAccessReq{Token: token, Id: profileID, Action: things.Viewer})
	if err != nil {
		return RulesPage{}, err
	}

	rules, err := rs.rules.RetrieveByProfile(ctx, profileID, pm)
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

func (rs *rulesService) ViewRule(ctx context.Context, token, id string) (Rule, error) {
	rule, err := rs.rules.RetrieveByID(ctx, id)
	if err != nil {
		return Rule{}, err
	}
	//TODO Replace by CanUserAccessGroup
	if _, err := rs.things.CanUserAccessProfile(ctx, &protomfx.UserAccessReq{Token: token, Id: rule.ProfileID, Action: things.Viewer}); err != nil {
		return Rule{}, err
	}

	return rule, nil
}

func (rs *rulesService) UpdateRule(ctx context.Context, token string, rule Rule) error {
	r, err := rs.rules.RetrieveByID(ctx, rule.ID)
	if err != nil {
		return err
	}
	//TODO Replace by CanUserAccessGroup
	if _, err := rs.things.CanUserAccessProfile(ctx, &protomfx.UserAccessReq{Token: token, Id: r.ProfileID, Action: things.Editor}); err != nil {
		return err
	}

	return rs.rules.Update(ctx, rule)
}

func (rs *rulesService) RemoveRules(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		webhook, err := rs.rules.RetrieveByID(ctx, id)
		if err != nil {
			return err
		}
		//TODO Replace by CanUserAccessGroup
		if _, err := rs.things.CanUserAccessProfile(ctx, &protomfx.UserAccessReq{Token: token, Id: webhook.ProfileID, Action: things.Editor}); err != nil {
			return errors.Wrap(errors.ErrAuthorization, err)
		}
	}
	return rs.rules.Remove(ctx, ids...)
}

func (rs *rulesService) Publish(ctx context.Context, message protomfx.Message) error {
	for _, rule := range message.Rules {
		if len(rule.Actions) == 0 {
			continue
		}

		isValid, payloads, err := processPayload(message.Payload, *rule, message.ContentType)
		if err != nil {
			return err
		}
		if isValid {
			continue
		}

		for _, action := range rule.Actions {
			for _, payload := range payloads {
				newMsg := message
				newMsg.Rules = []*protomfx.Rule{rule}
				newMsg.Payload = payload
				newMsg.Subject = action.Type

				switch action.Type {
				case actionTypeSMTP, actionTypeSMPP:
					if action.Id != "" {
						if err := rs.publisher.Publish(newMsg); err != nil {
							return err
						}
					}
					return errInvalidActionID
				case actionTypeAlarm:
					if err := rs.publisher.Publish(newMsg); err != nil {
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
