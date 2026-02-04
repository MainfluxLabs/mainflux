// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package rules

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/MainfluxLabs/mainflux/consumers"
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

	// CreateScripts persists multiple Lua scripts.
	CreateScripts(ctx context.Context, token, groupID string, scripts ...LuaScript) ([]LuaScript, error)

	// ListScriptsByThing retrieves a list of Scripts associated with a specific Thing.
	ListScriptsByThing(ctx context.Context, token, thingID string, pm apiutil.PageMetadata) (LuaScriptsPage, error)

	// ListScriptsByGroup retrieves a list of scripts belonging to a specific Group.
	ListScriptsByGroup(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (LuaScriptsPage, error)

	// ListThingIDsByScript retrieves a list of IDs of Things associated with a specific Script.
	ListThingIDsByScript(ctx context.Context, token, scriptID string) ([]string, error)

	// ViewScript retrieves a specific Script by its ID.
	ViewScript(ctx context.Context, token, id string) (LuaScript, error)

	// UpdateScript updates the Script identified by the provided ID.
	UpdateScript(ctx context.Context, token string, script LuaScript) error

	// RemoveScripts removes the Scripts identified by the provided IDs.
	RemoveScripts(ctx context.Context, token string, ids ...string) error

	// AssignScripts assigns one or more Scripts to a specific Thing.
	AssignScripts(ctx context.Context, token, thingID string, scriptIDs ...string) error

	// UnassignScripts unassigns one or omre scripts from a specific Thing.
	UnassignScripts(ctx context.Context, token, thingID string, scriptIDs ...string) error

	// ListScriptRunsByThing retrieves a list of Script Runs associated with a specific Thing.
	ListScriptRunsByThing(ctx context.Context, token, thingID string, pm apiutil.PageMetadata) (ScriptRunsPage, error)

	// RemoveScriptRuns removes the Runs identified by the provided IDs.
	RemoveScriptRuns(ctx context.Context, token string, ids ...string) error

	consumers.Consumer
}

const (
	// subjectAlarms represents subject used to publish messages that trigger an alarm.
	subjectAlarms = "alarms"
	// subjectSMTP represents subject used to publish messages that trigger an SMTP notification.
	subjectSMTP = "smtp"
	// subjectSMPP represents subject used to publish messages that trigger an SMPP notification.
	subjectSMPP = "smpp"
)

type rulesService struct {
	rules          RuleRepository
	things         protomfx.ThingsServiceClient
	pubsub         messaging.PubSub
	idProvider     uuid.IDProvider
	logger         logger.Logger
	scriptsEnabled bool
}

var _ Service = (*rulesService)(nil)

// New instantiates the rules service implementation.
func New(rules RuleRepository, things protomfx.ThingsServiceClient, pubsub messaging.PubSub, idp uuid.IDProvider, logger logger.Logger, scriptsEnabled bool) Service {
	return &rulesService{
		rules:          rules,
		things:         things,
		pubsub:         pubsub,
		idProvider:     idp,
		logger:         logger,
		scriptsEnabled: scriptsEnabled,
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

func (rs *rulesService) CreateScripts(ctx context.Context, token, groupID string, scripts ...LuaScript) ([]LuaScript, error) {
	_, err := rs.things.CanUserAccessGroup(ctx, &protomfx.UserAccessReq{Token: token, Id: groupID, Action: things.Editor})
	if err != nil {
		return []LuaScript{}, err
	}

	for i := range scripts {
		scripts[i].GroupID = groupID

		id, err := rs.idProvider.ID()
		if err != nil {
			return []LuaScript{}, err
		}
		scripts[i].ID = id
	}

	return rs.rules.SaveScripts(ctx, scripts...)
}

func (rs *rulesService) ListScriptsByThing(ctx context.Context, token, thingID string, pm apiutil.PageMetadata) (LuaScriptsPage, error) {
	_, err := rs.things.CanUserAccessThing(ctx, &protomfx.UserAccessReq{Token: token, Id: thingID, Action: things.Viewer})
	if err != nil {
		return LuaScriptsPage{}, err
	}

	return rs.rules.RetrieveScriptsByThing(ctx, thingID, pm)
}

func (rs *rulesService) ListScriptsByGroup(ctx context.Context, token, groupID string, pm apiutil.PageMetadata) (LuaScriptsPage, error) {
	_, err := rs.things.CanUserAccessGroup(ctx, &protomfx.UserAccessReq{Token: token, Id: groupID, Action: things.Viewer})
	if err != nil {
		return LuaScriptsPage{}, err
	}

	return rs.rules.RetrieveScriptsByGroup(ctx, groupID, pm)
}

func (rs *rulesService) ListThingIDsByScript(ctx context.Context, token, scriptID string) ([]string, error) {
	script, err := rs.rules.RetrieveScriptByID(ctx, scriptID)
	if err != nil {
		return []string{}, err
	}

	if _, err := rs.things.CanUserAccessGroup(ctx, &protomfx.UserAccessReq{Token: token, Id: script.GroupID, Action: things.Viewer}); err != nil {
		return []string{}, err
	}

	return rs.rules.RetrieveThingIDsByScript(ctx, scriptID)
}

func (rs *rulesService) ViewScript(ctx context.Context, token, id string) (LuaScript, error) {
	script, err := rs.rules.RetrieveScriptByID(ctx, id)
	if err != nil {
		return LuaScript{}, err
	}

	if _, err := rs.things.CanUserAccessGroup(ctx, &protomfx.UserAccessReq{Token: token, Id: script.GroupID, Action: things.Viewer}); err != nil {
		return LuaScript{}, err
	}

	return script, nil
}

func (rs *rulesService) UpdateScript(ctx context.Context, token string, script LuaScript) error {
	existingScript, err := rs.rules.RetrieveScriptByID(ctx, script.ID)
	if err != nil {
		return err
	}

	req := &protomfx.UserAccessReq{Token: token, Id: existingScript.GroupID, Action: things.Editor}
	if _, err := rs.things.CanUserAccessGroup(ctx, req); err != nil {
		return err
	}

	return rs.rules.UpdateScript(ctx, script)
}

func (rs *rulesService) RemoveScripts(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		script, err := rs.rules.RetrieveScriptByID(ctx, id)
		if err != nil {
			return err
		}

		if _, err := rs.things.CanUserAccessGroup(ctx, &protomfx.UserAccessReq{Token: token, Id: script.GroupID, Action: things.Editor}); err != nil {
			return err
		}
	}

	return rs.rules.RemoveScripts(ctx, ids...)
}

func (rs *rulesService) AssignScripts(ctx context.Context, token, thingID string, scriptIDs ...string) error {
	if _, err := rs.things.CanUserAccessThing(ctx, &protomfx.UserAccessReq{Token: token, Id: thingID, Action: things.Editor}); err != nil {
		return err
	}

	grID, err := rs.things.GetGroupIDByThingID(ctx, &protomfx.ThingID{Value: thingID})
	if err != nil {
		return err
	}

	for _, id := range scriptIDs {
		script, err := rs.rules.RetrieveScriptByID(ctx, id)
		if err != nil {
			return err
		}

		if script.GroupID != grID.GetValue() {
			return errors.ErrAuthorization
		}
	}

	if err := rs.rules.AssignScripts(ctx, thingID, scriptIDs...); err != nil {
		return err
	}

	return nil
}

func (rs *rulesService) UnassignScripts(ctx context.Context, token, thingID string, scriptIDs ...string) error {
	if _, err := rs.things.CanUserAccessThing(ctx, &protomfx.UserAccessReq{Token: token, Id: thingID, Action: things.Editor}); err != nil {
		return err
	}

	grID, err := rs.things.GetGroupIDByThingID(ctx, &protomfx.ThingID{Value: thingID})
	if err != nil {
		return err
	}

	for _, id := range scriptIDs {
		script, err := rs.rules.RetrieveScriptByID(ctx, id)
		if err != nil {
			return err
		}

		if script.GroupID != grID.GetValue() {
			return errors.ErrAuthorization
		}
	}

	if err := rs.rules.UnassignScripts(ctx, thingID, scriptIDs...); err != nil {
		return err
	}

	return nil
}

func (rs *rulesService) ListScriptRunsByThing(ctx context.Context, token, thingID string, pm apiutil.PageMetadata) (ScriptRunsPage, error) {
	if _, err := rs.things.CanUserAccessThing(ctx, &protomfx.UserAccessReq{Token: token, Id: thingID, Action: things.Viewer}); err != nil {
		return ScriptRunsPage{}, err
	}

	return rs.rules.RetrieveScriptRunsByThing(ctx, thingID, pm)
}

func (rs *rulesService) RemoveScriptRuns(ctx context.Context, token string, ids ...string) error {
	for _, id := range ids {
		run, err := rs.rules.RetrieveScriptRunByID(ctx, id)
		if err != nil {
			return err
		}

		if _, err := rs.things.CanUserAccessThing(ctx, &protomfx.UserAccessReq{Token: token, Id: run.ThingID, Action: things.Editor}); err != nil {
			return err
		}
	}

	return rs.rules.RemoveScriptRuns(ctx, ids...)
}

func (rs *rulesService) Consume(message any) error {
	ctx := context.Background()

	msg, ok := message.(protomfx.Message)
	if !ok {
		return errors.ErrMessage
	}

	var payload any
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return err
	}

	rulesPage, err := rs.rules.RetrieveByThing(ctx, msg.Publisher, apiutil.PageMetadata{})
	if err != nil {
		return err
	}

	// Process simple rules assigned to Publisher
	for _, rule := range rulesPage.Rules {
		if err := rs.processRule(&msg, payload, rule); err != nil {
			rs.logger.Error(fmt.Sprintf("processing rule with id %s failed with error: %v", rule.ID, err))
		}
	}

	if !rs.scriptsEnabled {
		return nil
	}

	scriptsPage, err := rs.rules.RetrieveScriptsByThing(ctx, msg.Publisher, apiutil.PageMetadata{})
	if err != nil {
		return err
	}

	// Process Lua scripts assigned to Publisher
	rs.processLuaScripts(ctx, &msg, payload, scriptsPage.Scripts...)

	return nil
}

type RuleRepository interface {
	// Save persists multiple rules. Rules are saved using a transaction.
	// If one rule fails then none will be saved.
	// Successful operation is indicated by a non-nil error response.
	Save(ctx context.Context, rules ...Rule) ([]Rule, error)

	// RetrieveByID retrieves a rule having the provided ID.
	RetrieveByID(ctx context.Context, id string) (Rule, error)

	// RetrieveByThing retrieves rules assigned to a certain thing,
	// identified by a given thing ID.
	RetrieveByThing(ctx context.Context, thingID string, pm apiutil.PageMetadata) (RulesPage, error)

	// RetrieveByGroup retrieves rules related to a certain group,
	// identified by a given group ID.
	RetrieveByGroup(ctx context.Context, groupID string, pm apiutil.PageMetadata) (RulesPage, error)

	// RetrieveThingIDsByRule retrieves all thing IDs that have the given rule assigned.
	RetrieveThingIDsByRule(ctx context.Context, ruleID string) ([]string, error)

	// Update performs an update to the existing rule.
	// A non-nil error is returned to indicate operation failure.
	Update(ctx context.Context, r Rule) error

	// Remove removes rules having the provided IDs.
	Remove(ctx context.Context, ids ...string) error

	// RemoveByGroup removes rules related to a certain group,
	// identified by a given group ID.
	RemoveByGroup(ctx context.Context, groupID string) error

	// Assign assigns rules to the specified thing.
	Assign(ctx context.Context, thingID string, ruleIDs ...string) error

	// Unassign removes specific rule assignments from a given thing.
	Unassign(ctx context.Context, thingID string, ruleIDs ...string) error

	// UnassignByThing removes all rule assignments for a certain thing,
	// identified by a given thing ID.
	UnassignByThing(ctx context.Context, thingID string) error

	// SaveScripts persists multiple Lua scripts.
	SaveScripts(ctx context.Context, scripts ...LuaScript) ([]LuaScript, error)

	// RetrieveScriptByID retrieves a single Lua script denoted by ID.
	RetrieveScriptByID(ctx context.Context, id string) (LuaScript, error)

	// RetrieveScriptsByThing retrieves a list of Lua scripts assigned to a specific Thing.
	RetrieveScriptsByThing(ctx context.Context, thingID string, pm apiutil.PageMetadata) (LuaScriptsPage, error)

	// RetrieveScriptsByGroup retrieves a list of Lua scripts belonging to a specific Group.
	RetrieveScriptsByGroup(ctx context.Context, groupID string, pm apiutil.PageMetadata) (LuaScriptsPage, error)

	//R etrieveThingIDsByScript retrieves a list of Thing IDs to which the specific Lua script is assigned.
	RetrieveThingIDsByScript(ctx context.Context, scriptID string) ([]string, error)

	// UpdateScript updates the script denoted by script.ID.
	UpdateScript(ctx context.Context, script LuaScript) error

	// RemoveScripts removes Lua scripts with the provided ids.
	RemoveScripts(ctx context.Context, ids ...string) error

	// AssignScripts assigns one or more Lua scripts to a specific Thing.
	AssignScripts(ctx context.Context, thingID string, scriptIDs ...string) error

	// Unassign unassgins one or omre Lua scripts from a specific Thing.
	UnassignScripts(ctx context.Context, thingID string, scriptIDs ...string) error

	// SaveScriptRuns preserves multiple ScriptRuns.
	SaveScriptRuns(ctx context.Context, runs ...ScriptRun) ([]ScriptRun, error)

	// RetrieveScriptRunByID retrieves a single ScriptRun based on its ID.
	RetrieveScriptRunByID(ctx context.Context, id string) (ScriptRun, error)

	// RetrieveScriptRunsByThing retrieves a list of Script runs by Thing ID.
	RetrieveScriptRunsByThing(ctx context.Context, thingID string, pm apiutil.PageMetadata) (ScriptRunsPage, error)

	// RemoveScriptRuns removes one or more Script runs by IDs.
	RemoveScriptRuns(ctx context.Context, ids ...string) error
}
