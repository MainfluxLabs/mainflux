// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package rules_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	authmock "github.com/MainfluxLabs/mainflux/pkg/mocks"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/rules"
	"github.com/MainfluxLabs/mainflux/rules/mocks"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	token      = "admin@example.com"
	wrongValue = "wrong-value"
	thingID    = "5384fb1c-d0ae-4cbe-be52-c54223150fe0"
	otherThing = "6384fb1c-d0ae-4cbe-be52-c54223150fe1"
	groupID    = "574106f7-030e-4881-8ab0-151195c29f94"
	subject    = "things." + thingID + ".messages"
)

func threshold(v float64) *float64 { return &v }

func newService() rules.Service {
	return newServiceWithPub(mocks.NewPublisher())
}

func newServiceWithPub(pub rules.Publisher) rules.Service {
	return newServiceWithRepo(mocks.NewRuleRepository(), pub)
}

func newServiceWithRepo(rulesRepo rules.Repository, pub rules.Publisher) rules.Service {
	ths := authmock.NewThingsServiceClient(
		nil,
		map[string]things.Thing{
			token:      {ID: thingID, GroupID: groupID},
			thingID:    {ID: thingID, GroupID: groupID},
			otherThing: {ID: otherThing, GroupID: wrongValue},
		},
		map[string]things.Group{
			token: {ID: groupID},
		},
	)

	idp := uuid.NewMock()
	log := logger.NewMock()

	return rules.New(rulesRepo, ths, pub, idp, log)
}

func saveRules(t *testing.T, svc rules.Service, n int) []rules.Rule {
	t.Helper()

	var rs []rules.Rule
	for i := range n {
		rs = append(rs, rules.Rule{
			Name:        fmt.Sprintf("rule-%d", i),
			Description: fmt.Sprintf("desc-%d", i),
			Input:       rules.Input{Type: rules.InputTypeMessage, ThingIDs: []string{thingID}},
			Conditions: []rules.Condition{
				{Field: "temperature", Comparator: ">", Threshold: threshold(25)},
			},
			Operator: rules.OperatorAND,
			Actions:  []rules.Action{{ID: "action-1", Type: rules.ActionTypeAlarm}},
		})
	}

	saved, err := svc.CreateRules(context.Background(), token, groupID, rs...)
	require.Nil(t, err)
	require.Len(t, saved, n)

	return saved
}

func toRecords(t *testing.T, payloads ...any) []*protomfx.Record {
	t.Helper()

	var records []*protomfx.Record
	for _, payload := range payloads {
		b, err := json.Marshal(payload)
		require.Nil(t, err)
		records = append(records, &protomfx.Record{Payload: b})
	}

	return records
}

func TestConsumeMessage(t *testing.T) {
	defaultConditions := []rules.Condition{
		{Field: "temperature", Comparator: ">", Threshold: threshold(25)},
	}

	cases := []struct {
		desc       string
		conditions []rules.Condition
		operator   string
		msg        protomfx.Message
		err        error
	}{
		{
			desc:       "valid JSON message triggers rule",
			conditions: defaultConditions,
			operator:   rules.OperatorAND,
			msg: protomfx.Message{
				ThingId:     thingID,
				Records:     toRecords(t, map[string]any{"temperature": float64(30)}),
				ContentType: "application/json",
			},
			err: nil,
		},
		{
			desc:       "JSON message below threshold does not trigger",
			conditions: defaultConditions,
			operator:   rules.OperatorAND,
			msg: protomfx.Message{
				ThingId:     thingID,
				Records:     toRecords(t, map[string]any{"temperature": float64(20)}),
				ContentType: "application/json",
			},
			err: nil,
		},
		{
			desc:       "JSON array payload with partial match",
			conditions: defaultConditions,
			operator:   rules.OperatorAND,
			msg: protomfx.Message{
				ThingId: thingID,
				Records: toRecords(t,
					map[string]any{"temperature": float64(30)},
					map[string]any{"temperature": float64(10)},
				),
				ContentType: "application/json",
			},
			err: nil,
		},
		{
			desc:       "SenML payload triggers rule",
			conditions: defaultConditions,
			operator:   rules.OperatorAND,
			msg: protomfx.Message{
				ThingId: thingID,
				Records: toRecords(t,
					map[string]any{"name": "temperature", "value": float64(30)},
				),
				ContentType: "application/senml+json",
			},
			err: nil,
		},
		// AND / OR operators
		{
			desc: "AND operator: all conditions met",
			conditions: []rules.Condition{
				{Field: "temperature", Comparator: ">", Threshold: threshold(20)},
				{Field: "humidity", Comparator: "<", Threshold: threshold(30)},
			},
			operator: rules.OperatorAND,
			msg: protomfx.Message{
				ThingId: thingID,
				Records: toRecords(t, map[string]any{
					"temperature": float64(25),
					"humidity":    float64(10),
				}),
				ContentType: "application/json",
			},
			err: nil,
		},
		{
			desc: "AND operator: one condition not met",
			conditions: []rules.Condition{
				{Field: "temperature", Comparator: ">", Threshold: threshold(30)},
				{Field: "humidity", Comparator: "<", Threshold: threshold(30)},
			},
			operator: rules.OperatorAND,
			msg: protomfx.Message{
				ThingId: thingID,
				Records: toRecords(t, map[string]any{
					"temperature": float64(25),
					"humidity":    float64(10),
				}),
				ContentType: "application/json",
			},
			err: nil,
		},
		{
			desc: "OR operator: one condition met",
			conditions: []rules.Condition{
				{Field: "temperature", Comparator: ">", Threshold: threshold(30)},
				{Field: "humidity", Comparator: "<", Threshold: threshold(30)},
			},
			operator: rules.OperatorOR,
			msg: protomfx.Message{
				ThingId: thingID,
				Records: toRecords(t, map[string]any{
					"temperature": float64(25),
					"humidity":    float64(10),
				}),
				ContentType: "application/json",
			},
			err: nil,
		},
		{
			desc: "OR operator: no condition met",
			conditions: []rules.Condition{
				{Field: "temperature", Comparator: ">", Threshold: threshold(30)},
				{Field: "humidity", Comparator: "<", Threshold: threshold(20)},
			},
			operator: rules.OperatorOR,
			msg: protomfx.Message{
				ThingId: thingID,
				Records: toRecords(t, map[string]any{
					"temperature": float64(25),
					"humidity":    float64(25),
				}),
				ContentType: "application/json",
			},
			err: nil,
		},
		// comparators
		{
			desc: "== comparator: exact match",
			conditions: []rules.Condition{
				{Field: "temperature", Comparator: "==", Threshold: threshold(25)},
			},
			operator: rules.OperatorAND,
			msg: protomfx.Message{
				ThingId:     thingID,
				Records:     toRecords(t, map[string]any{"temperature": float64(25)}),
				ContentType: "application/json",
			},
			err: nil,
		},
		{
			desc: ">= comparator: equal value",
			conditions: []rules.Condition{
				{Field: "temperature", Comparator: ">=", Threshold: threshold(25)},
			},
			operator: rules.OperatorAND,
			msg: protomfx.Message{
				ThingId:     thingID,
				Records:     toRecords(t, map[string]any{"temperature": float64(25)}),
				ContentType: "application/json",
			},
			err: nil,
		},
		{
			desc: "<= comparator: equal value",
			conditions: []rules.Condition{
				{Field: "temperature", Comparator: "<=", Threshold: threshold(25)},
			},
			operator: rules.OperatorAND,
			msg: protomfx.Message{
				ThingId:     thingID,
				Records:     toRecords(t, map[string]any{"temperature": float64(25)}),
				ContentType: "application/json",
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		svc := newService()

		if len(tc.conditions) > 0 {
			_, err := svc.CreateRules(context.Background(), token, groupID, rules.Rule{
				Name:       "test-rule",
				Input:      rules.Input{Type: rules.InputTypeMessage, ThingIDs: []string{thingID}},
				Conditions: tc.conditions,
				Operator:   tc.operator,
				Actions:    []rules.Action{{Type: rules.ActionTypeAlarm}},
			})
			require.Nil(t, err)
		}

		err := svc.ConsumeMessage(subject, tc.msg)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}

func TestCreateRules(t *testing.T) {
	svc := newService()

	rule := rules.Rule{
		Name:        "temp-rule",
		Description: "triggers on high temperature",
		Conditions: []rules.Condition{
			{Field: "temperature", Comparator: ">", Threshold: threshold(30)},
		},
		Operator: rules.OperatorAND,
		Actions:  []rules.Action{{Type: rules.ActionTypeAlarm}},
	}

	thingRule := rule
	thingRule.Input = rules.Input{Type: rules.InputTypeMessage, ThingIDs: []string{thingID}}

	otherThingRule := rule
	otherThingRule.Input = rules.Input{Type: rules.InputTypeMessage, ThingIDs: []string{otherThing}}

	cases := []struct {
		desc    string
		token   string
		groupID string
		rules   []rules.Rule
		err     error
	}{
		{
			desc:    "create rules with thing of the group",
			token:   token,
			groupID: groupID,
			rules:   []rules.Rule{thingRule},
			err:     nil,
		},
		{
			desc:    "create rules with thing of another group",
			token:   token,
			groupID: groupID,
			rules:   []rules.Rule{otherThingRule},
			err:     errors.ErrAuthorization,
		},
		{
			desc:    "create rules with valid token",
			token:   token,
			groupID: groupID,
			rules:   []rules.Rule{rule},
			err:     nil,
		},
		{
			desc:    "create rules with invalid token",
			token:   wrongValue,
			groupID: groupID,
			rules:   []rules.Rule{rule},
			err:     errors.ErrAuthentication,
		},
		{
			desc:    "create rules with invalid group ID",
			token:   token,
			groupID: wrongValue,
			rules:   []rules.Rule{rule},
			err:     errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		_, err := svc.CreateRules(context.Background(), tc.token, tc.groupID, tc.rules...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}

func TestListRulesByGroup(t *testing.T) {
	svc := newService()
	n := 10
	saveRules(t, svc, n)

	cases := []struct {
		desc         string
		token        string
		groupID      string
		pageMetadata rules.PageMetadata
		size         uint64
		err          error
	}{
		{
			desc:    "list rules by group",
			token:   token,
			groupID: groupID,
			pageMetadata: rules.PageMetadata{
				Offset: 0,
				Limit:  uint64(n),
			},
			size: uint64(n),
			err:  nil,
		},
		{
			desc:    "list rules by group with no limit",
			token:   token,
			groupID: groupID,
			pageMetadata: rules.PageMetadata{
				Limit: 0,
			},
			size: uint64(n),
			err:  nil,
		},
		{
			desc:    "list last rule by group",
			token:   token,
			groupID: groupID,
			pageMetadata: rules.PageMetadata{
				Offset: uint64(n) - 1,
				Limit:  uint64(n),
			},
			size: 1,
			err:  nil,
		},
		{
			desc:    "list empty set of rules by group",
			token:   token,
			groupID: groupID,
			pageMetadata: rules.PageMetadata{
				Offset: uint64(n) + 1,
				Limit:  uint64(n),
			},
			size: 0,
			err:  nil,
		},
		{
			desc:    "list rules by group with invalid auth token",
			token:   wrongValue,
			groupID: groupID,
			pageMetadata: rules.PageMetadata{
				Offset: 0,
				Limit:  uint64(n),
			},
			size: 0,
			err:  errors.ErrAuthentication,
		},
		{
			desc:    "list rules by group with invalid group ID",
			token:   token,
			groupID: wrongValue,
			pageMetadata: rules.PageMetadata{
				Offset: 0,
				Limit:  uint64(n),
			},
			size: 0,
			err:  errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		page, err := svc.ListRulesByGroup(context.Background(), tc.token, tc.groupID, tc.pageMetadata)
		size := uint64(len(page.Rules))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d", tc.desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}

func TestListRulesByThing(t *testing.T) {
	svc := newService()
	n := 10
	saveRules(t, svc, n)

	cases := []struct {
		desc         string
		token        string
		thingID      string
		pageMetadata rules.PageMetadata
		size         uint64
		err          error
	}{
		{
			desc:    "list rules by thing",
			token:   token,
			thingID: thingID,
			pageMetadata: rules.PageMetadata{
				Offset: 0,
				Limit:  uint64(n),
			},
			size: uint64(n),
			err:  nil,
		},
		{
			desc:    "list rules by thing with no limit",
			token:   token,
			thingID: thingID,
			pageMetadata: rules.PageMetadata{
				Limit: 0,
			},
			size: uint64(n),
			err:  nil,
		},
		{
			desc:    "list last rule by thing",
			token:   token,
			thingID: thingID,
			pageMetadata: rules.PageMetadata{
				Offset: uint64(n) - 1,
				Limit:  uint64(n),
			},
			size: 1,
			err:  nil,
		},
		{
			desc:    "list empty set of rules by thing",
			token:   token,
			thingID: thingID,
			pageMetadata: rules.PageMetadata{
				Offset: uint64(n) + 1,
				Limit:  uint64(n),
			},
			size: 0,
			err:  nil,
		},
		{
			desc:    "list rules by thing with invalid auth token",
			token:   wrongValue,
			thingID: thingID,
			pageMetadata: rules.PageMetadata{
				Offset: 0,
				Limit:  uint64(n),
			},
			size: 0,
			err:  errors.ErrAuthentication,
		},
		{
			desc:    "list rules by thing with invalid thing ID",
			token:   token,
			thingID: wrongValue,
			pageMetadata: rules.PageMetadata{
				Offset: 0,
				Limit:  uint64(n),
			},
			size: 0,
			err:  errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		page, err := svc.ListRulesByThing(context.Background(), tc.token, tc.thingID, tc.pageMetadata)
		size := uint64(len(page.Rules))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%s: expected size %d got %d", tc.desc, tc.size, size))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}

func TestListThingIDsByRule(t *testing.T) {
	svc := newService()
	saved := saveRules(t, svc, 1)
	ruleID := saved[0].ID

	cases := []struct {
		desc   string
		token  string
		ruleID string
		size   int
		err    error
	}{
		{
			desc:   "list thing IDs by rule",
			token:  token,
			ruleID: ruleID,
			size:   1,
			err:    nil,
		},
		{
			desc:   "list thing IDs by rule with invalid token",
			token:  wrongValue,
			ruleID: ruleID,
			size:   0,
			err:    errors.ErrAuthentication,
		},
		{
			desc:   "list thing IDs by non-existing rule",
			token:  token,
			ruleID: wrongValue,
			size:   0,
			err:    dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		ids, err := svc.ListThingIDsByRule(context.Background(), tc.token, tc.ruleID)
		assert.Equal(t, tc.size, len(ids), fmt.Sprintf("%s: expected size %d got %d", tc.desc, tc.size, len(ids)))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}

func TestViewRule(t *testing.T) {
	svc := newService()
	saved := saveRules(t, svc, 1)
	ruleID := saved[0].ID

	cases := []struct {
		desc   string
		token  string
		ruleID string
		err    error
	}{
		{
			desc:   "view existing rule",
			token:  token,
			ruleID: ruleID,
			err:    nil,
		},
		{
			desc:   "view rule with invalid token",
			token:  wrongValue,
			ruleID: ruleID,
			err:    errors.ErrAuthentication,
		},
		{
			desc:   "view non-existing rule",
			token:  token,
			ruleID: wrongValue,
			err:    dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		res, err := svc.ViewRule(context.Background(), tc.token, tc.ruleID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
		if err == nil {
			assert.Equal(t, saved[0], res, fmt.Sprintf("%s: expected rule %+v got %+v", tc.desc, saved[0], res))
		}
	}
}

func TestUpdateRule(t *testing.T) {
	svc := newService()
	saved := saveRules(t, svc, 1)
	ruleID := saved[0].ID

	updated := rules.Rule{
		ID:          ruleID,
		GroupID:     groupID,
		Name:        "updated-rule",
		Description: "updated description",
		Conditions: []rules.Condition{
			{Field: "humidity", Comparator: "<", Threshold: threshold(50)},
		},
		Operator: rules.OperatorOR,
		Actions:  []rules.Action{{ID: "action-2", Type: rules.ActionTypeSMTP}},
	}

	cases := []struct {
		desc  string
		token string
		rule  rules.Rule
		err   error
	}{
		{
			desc:  "update existing rule",
			token: token,
			rule:  updated,
			err:   nil,
		},
		{
			desc:  "update rule with invalid token",
			token: wrongValue,
			rule:  updated,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "update non-existing rule",
			token: token,
			rule:  rules.Rule{ID: wrongValue, Name: "no-rule"},
			err:   dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.UpdateRule(context.Background(), tc.token, tc.rule)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}

func TestRemoveRules(t *testing.T) {
	svc := newService()
	saved := saveRules(t, svc, 1)
	ruleID := saved[0].ID

	cases := []struct {
		desc  string
		token string
		ids   []string
		err   error
	}{
		{
			desc:  "remove rules with invalid token",
			token: wrongValue,
			ids:   []string{ruleID},
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "remove existing rule",
			token: token,
			ids:   []string{ruleID},
			err:   nil,
		},
		{
			desc:  "remove non-existing rule",
			token: token,
			ids:   []string{wrongValue},
			err:   dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveRules(context.Background(), tc.token, tc.ids...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}

func TestRemoveRulesByGroup(t *testing.T) {
	svc := newService()
	saveRules(t, svc, 3)

	cases := []struct {
		desc    string
		groupID string
		err     error
	}{
		{
			desc:    "remove rules by group",
			groupID: groupID,
			err:     nil,
		},
		{
			desc:    "remove rules by non-existing group",
			groupID: wrongValue,
			err:     nil,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveRulesByGroup(context.Background(), tc.groupID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}
func TestUnassignRulesFromThing(t *testing.T) {
	svc := newService()
	n := 3
	saveRules(t, svc, n)

	cases := []struct {
		desc    string
		thingID string
		err     error
	}{
		{
			desc:    "unassign all rules from thing",
			thingID: thingID,
			err:     nil,
		},
		{
			desc:    "unassign rules from non-existing thing",
			thingID: wrongValue,
			err:     nil,
		},
	}

	for _, tc := range cases {
		err := svc.UnassignRulesFromThing(context.Background(), tc.thingID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}

func TestConsumeMessageScriptLogLimit(t *testing.T) {
	svc := newService()

	script := rules.LuaScript{
		Name:   "log-flood",
		Script: "for i = 1, 300 do mfx.log('line') end return true",
	}
	scripts, err := svc.CreateScripts(context.Background(), token, groupID, script)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	rule := rules.Rule{
		Name:       "script-rule",
		Input:      rules.Input{Type: rules.InputTypeMessage, ThingIDs: []string{thingID}},
		Conditions: []rules.Condition{{Type: rules.ConditionTypeScript, ScriptID: scripts[0].ID}},
		Actions:    []rules.Action{{Type: rules.ActionTypeAlarm, Level: 1}},
	}
	_, err = svc.CreateRules(context.Background(), token, groupID, rule)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	msg := protomfx.Message{
		ThingId:     thingID,
		Records:     toRecords(t, map[string]any{"temperature": float64(30)}),
		ContentType: "application/json",
	}
	err = svc.ConsumeMessage(subject, msg)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	pm := rules.PageMetadata{Limit: 10}
	page, err := svc.ListRunsByThing(context.Background(), token, thingID, pm)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	require.Len(t, page.Runs, 1)

	logs := len(page.Runs[0].Details.Conditions[0].Logs)
	assert.Equal(t, 256, logs, fmt.Sprintf("expected 256 log lines got %d\n", logs))
}

func TestConsumeMessageScriptSandbox(t *testing.T) {
	svc := newService()

	script := rules.LuaScript{
		Name:   "sandbox",
		Script: "mfx.log(type(dofile)..' '..type(loadfile)..' '..type(load)) return true",
	}
	scripts, err := svc.CreateScripts(context.Background(), token, groupID, script)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	rule := rules.Rule{
		Name:       "sandbox-rule",
		Input:      rules.Input{Type: rules.InputTypeMessage, ThingIDs: []string{thingID}},
		Conditions: []rules.Condition{{Type: rules.ConditionTypeScript, ScriptID: scripts[0].ID}},
		Actions:    []rules.Action{{Type: rules.ActionTypeAlarm, Level: 1}},
	}
	_, err = svc.CreateRules(context.Background(), token, groupID, rule)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	msg := protomfx.Message{
		ThingId:     thingID,
		Records:     toRecords(t, map[string]any{"temperature": float64(30)}),
		ContentType: "application/json",
	}
	err = svc.ConsumeMessage(subject, msg)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	pm := rules.PageMetadata{Limit: 10}
	page, err := svc.ListRunsByThing(context.Background(), token, thingID, pm)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	require.Len(t, page.Runs, 1)

	logs := page.Runs[0].Details.Conditions[0].Logs
	desc := fmt.Sprintf("expected file and chunk loaders to be unset got %v\n", logs)
	assert.Equal(t, []string{"nil nil nil"}, logs, desc)
}

func TestConsumeMessageScriptRuns(t *testing.T) {
	cases := []struct {
		desc   string
		script string
		total  uint64
		status string
	}{
		{
			desc:   "store run of script that returns true",
			script: "return true",
			total:  1,
			status: rules.RuleRunStatusSuccess,
		},
		{
			desc:   "store run of script that fails",
			script: "error('failure')",
			total:  1,
			status: rules.RuleRunStatusFail,
		},
		{
			desc:   "skip run of script that returns false",
			script: "return false",
			total:  0,
		},
		{
			desc:   "skip run of script that returns no value",
			script: "mfx.log('no value')",
			total:  0,
		},
	}

	for _, tc := range cases {
		svc := newService()

		script := rules.LuaScript{Name: "script", Script: tc.script}
		scripts, err := svc.CreateScripts(context.Background(), token, groupID, script)
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))

		rule := rules.Rule{
			Name:       "script-rule",
			Input:      rules.Input{Type: rules.InputTypeMessage, ThingIDs: []string{thingID}},
			Conditions: []rules.Condition{{Type: rules.ConditionTypeScript, ScriptID: scripts[0].ID}},
			Actions:    []rules.Action{{Type: rules.ActionTypeAlarm, Level: 1}},
		}
		_, err = svc.CreateRules(context.Background(), token, groupID, rule)
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))

		msg := protomfx.Message{
			ThingId:     thingID,
			Records:     toRecords(t, map[string]any{"temperature": float64(30)}),
			ContentType: "application/json",
		}
		err = svc.ConsumeMessage(subject, msg)
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))

		pm := rules.PageMetadata{Limit: 10}
		page, err := svc.ListRunsByThing(context.Background(), token, thingID, pm)
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))
		desc := fmt.Sprintf("%s: expected %d runs got %d\n", tc.desc, tc.total, page.Total)
		assert.Equal(t, tc.total, page.Total, desc)

		for _, run := range page.Runs {
			desc := fmt.Sprintf("%s: expected status %s got %s\n", tc.desc, tc.status, run.Status)
			assert.Equal(t, tc.status, run.Status, desc)
		}
	}
}

// runsRepository records the rule runs saved through the embedded repository.
type runsRepository struct {
	rules.Repository
	runs []rules.RuleRun
}

func (rr *runsRepository) SaveRuns(ctx context.Context, runs ...rules.RuleRun) ([]rules.RuleRun, error) {
	rr.runs = append(rr.runs, runs...)
	return rr.Repository.SaveRuns(ctx, runs...)
}

func TestConsumeRuns(t *testing.T) {
	met := rules.RuleRunCondition{Type: rules.ConditionTypeThreshold, Met: true}

	cases := []struct {
		desc    string
		input   string
		script  string
		value   any
		pub     rules.Publisher
		status  string
		details rules.RuleRunDetails
	}{
		{
			desc:   "store run of fired message rule",
			input:  rules.InputTypeMessage,
			value:  30,
			pub:    mocks.NewPublisher(),
			status: rules.RuleRunStatusSuccess,
			details: rules.RuleRunDetails{
				Conditions: []rules.RuleRunCondition{met},
				Actions:    []rules.RuleRunAction{{Type: rules.ActionTypeAlarm}},
			},
		},
		{
			desc:  "skip run of message rule that does not fire",
			input: rules.InputTypeMessage,
			value: 20,
			pub:   mocks.NewPublisher(),
		},
		{
			desc:   "store run of message rule with failed action",
			input:  rules.InputTypeMessage,
			value:  30,
			pub:    mocks.NewFailingPublisher(),
			status: rules.RuleRunStatusFail,
			details: rules.RuleRunDetails{
				Conditions: []rules.RuleRunCondition{met},
				Actions: []rules.RuleRunAction{{
					Type:  rules.ActionTypeAlarm,
					Error: messaging.ErrPublishAlarm.Error(),
				}},
			},
		},
		{
			desc:   "store run of message rule with failed threshold condition",
			input:  rules.InputTypeMessage,
			value:  "invalid",
			pub:    mocks.NewPublisher(),
			status: rules.RuleRunStatusFail,
			details: rules.RuleRunDetails{
				Conditions: []rules.RuleRunCondition{{
					Type:  rules.ConditionTypeThreshold,
					Error: `strconv.ParseFloat: parsing "invalid": invalid syntax`,
				}},
			},
		},
		{
			desc:   "store run of message rule fired by script condition",
			input:  rules.InputTypeMessage,
			script: "mfx.log('fired') return true",
			pub:    mocks.NewPublisher(),
			status: rules.RuleRunStatusSuccess,
			details: rules.RuleRunDetails{
				Conditions: []rules.RuleRunCondition{{
					Type:   rules.ConditionTypeScript,
					Met:    true,
					Result: true,
					Logs:   []string{"fired"},
				}},
				Actions: []rules.RuleRunAction{{Type: rules.ActionTypeAlarm}},
			},
		},
		{
			desc:   "store run of message rule with failed script condition",
			input:  rules.InputTypeMessage,
			script: "error('failure')",
			pub:    mocks.NewPublisher(),
			status: rules.RuleRunStatusFail,
			details: rules.RuleRunDetails{
				Conditions: []rules.RuleRunCondition{{
					Type:  rules.ConditionTypeScript,
					Logs:  []string{},
					Error: `runtime error: [string "error('failure')"]:1: failure`,
				}},
			},
		},
		{
			desc:   "store run of fired alarm rule",
			input:  rules.InputTypeAlarm,
			value:  30,
			pub:    mocks.NewPublisher(),
			status: rules.RuleRunStatusSuccess,
			details: rules.RuleRunDetails{
				Conditions: []rules.RuleRunCondition{met},
				Actions:    []rules.RuleRunAction{{Type: rules.ActionTypeWebhook}},
			},
		},
		{
			desc:  "skip run of alarm rule that does not fire",
			input: rules.InputTypeAlarm,
			value: 20,
			pub:   mocks.NewPublisher(),
		},
		{
			desc:   "store run of alarm rule with failed action",
			input:  rules.InputTypeAlarm,
			value:  30,
			pub:    mocks.NewFailingPublisher(),
			status: rules.RuleRunStatusFail,
			details: rules.RuleRunDetails{
				Conditions: []rules.RuleRunCondition{met},
				Actions: []rules.RuleRunAction{{
					Type:  rules.ActionTypeWebhook,
					Error: messaging.ErrPublishWebhook.Error(),
				}},
			},
		},
	}

	for _, tc := range cases {
		repo := &runsRepository{Repository: mocks.NewRuleRepository()}
		svc := newServiceWithRepo(repo, tc.pub)

		field, action := "temperature", rules.Action{Type: rules.ActionTypeAlarm}
		if tc.input == rules.InputTypeAlarm {
			field, action = "level", rules.Action{Type: rules.ActionTypeWebhook}
		}

		condition := rules.Condition{
			Type:       rules.ConditionTypeThreshold,
			Field:      field,
			Comparator: ">",
			Threshold:  threshold(25),
		}
		if tc.script != "" {
			script := rules.LuaScript{Name: "script", Script: tc.script}
			scripts, err := svc.CreateScripts(context.Background(), token, groupID, script)
			require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))
			condition = rules.Condition{Type: rules.ConditionTypeScript, ScriptID: scripts[0].ID}
			tc.details.Conditions[0].ScriptID = scripts[0].ID
		}

		rule := rules.Rule{
			Name:       "rule",
			Input:      rules.Input{Type: tc.input, ThingIDs: []string{thingID}},
			Conditions: []rules.Condition{condition},
			Operator:   rules.OperatorAND,
			Actions:    []rules.Action{action},
		}
		rls, err := svc.CreateRules(context.Background(), token, groupID, rule)
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))

		switch tc.input {
		case rules.InputTypeAlarm:
			err = svc.ConsumeAlarm(subject, protomfx.Alarm{ThingId: thingID, Level: int32(tc.value.(int))})
		default:
			err = svc.ConsumeMessage(subject, protomfx.Message{
				ThingId:     thingID,
				Records:     toRecords(t, map[string]any{field: tc.value}),
				ContentType: "application/json",
			})
		}
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))

		var want []rules.RuleRun
		if tc.status != "" {
			want = []rules.RuleRun{{
				RuleID:  rls[0].ID,
				ThingID: thingID,
				Status:  tc.status,
				Details: tc.details,
			}}
		}

		var got []rules.RuleRun
		for _, run := range repo.runs {
			got = append(got, rules.RuleRun{
				RuleID:  run.RuleID,
				ThingID: run.ThingID,
				Status:  run.Status,
				Details: run.Details,
			})
		}
		assert.Equal(t, want, got, fmt.Sprintf("%s: unexpected rule runs", tc.desc))
	}
}

func TestConsumeRunsScriptResult(t *testing.T) {
	cases := []struct {
		desc   string
		script string
		status string
		result rules.RuleRunCondition
	}{
		{
			desc:   "store object returned by script condition",
			script: "return {temperature = mfx.message.payload.temperature, unit = 'C'}",
			status: rules.RuleRunStatusSuccess,
			result: rules.RuleRunCondition{
				Result: map[string]any{"temperature": float64(30), "unit": "C"},
			},
		},
		{
			desc:   "store array of objects returned by script condition",
			script: "return {{value = 1}, {value = 2}}",
			status: rules.RuleRunStatusSuccess,
			result: rules.RuleRunCondition{
				Result: []any{map[string]any{"value": float64(1)}, map[string]any{"value": float64(2)}},
			},
		},
		{
			desc:   "store false returned by script condition",
			script: "return false",
			status: rules.RuleRunStatusSuccess,
			result: rules.RuleRunCondition{Result: false},
		},
		{
			desc:   "store no result of script condition without return value",
			script: "mfx.log('no value')",
			status: rules.RuleRunStatusSuccess,
			result: rules.RuleRunCondition{Logs: []string{"no value"}},
		},
		{
			desc:   "store error of script condition that returns cyclic table",
			script: "local t = {} t.self = t return t",
			status: rules.RuleRunStatusFail,
			result: rules.RuleRunCondition{Error: "script return value exceeds size limit"},
		},
		{
			desc:   "store error of script condition that returns repeatedly referenced tables",
			script: "local t = {} for i = 1, 30 do t = {t, t} end return t",
			status: rules.RuleRunStatusFail,
			result: rules.RuleRunCondition{Error: "script return value exceeds size limit"},
		},
		{
			desc:   "store error of script condition that returns oversized string",
			script: "return string.rep('a', 100000)",
			status: rules.RuleRunStatusFail,
			result: rules.RuleRunCondition{Error: "script return value exceeds size limit"},
		},
		{
			desc:   "store error of script condition that returns duplicate table keys",
			script: "return {[1] = 'a', ['1'] = 'b'}",
			status: rules.RuleRunStatusFail,
			result: rules.RuleRunCondition{
				Error: "script return value contains an unsupported or duplicate table key",
			},
		},
	}

	for _, tc := range cases {
		repo := &runsRepository{Repository: mocks.NewRuleRepository()}
		svc := newServiceWithRepo(repo, mocks.NewPublisher())

		script := rules.LuaScript{Name: "script", Script: tc.script}
		scripts, err := svc.CreateScripts(context.Background(), token, groupID, script)
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))

		rule := rules.Rule{
			Name:  "rule",
			Input: rules.Input{Type: rules.InputTypeMessage, ThingIDs: []string{thingID}},
			Conditions: []rules.Condition{
				{
					Type:       rules.ConditionTypeThreshold,
					Field:      "temperature",
					Comparator: ">",
					Threshold:  threshold(25),
				},
				{Type: rules.ConditionTypeScript, ScriptID: scripts[0].ID},
			},
			Operator: rules.OperatorOR,
			Actions:  []rules.Action{{Type: rules.ActionTypeAlarm}},
		}
		_, err = svc.CreateRules(context.Background(), token, groupID, rule)
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))

		err = svc.ConsumeMessage(subject, protomfx.Message{
			ThingId:     thingID,
			Records:     toRecords(t, map[string]any{"temperature": float64(30)}),
			ContentType: "application/json",
		})
		require.Nil(t, err, fmt.Sprintf("%s: unexpected error: %s", tc.desc, err))
		require.Len(t, repo.runs, 1, fmt.Sprintf("%s: expected one rule run", tc.desc))

		want := tc.result
		want.Type = rules.ConditionTypeScript
		want.ScriptID = scripts[0].ID
		want.Met = tc.result.Result == true
		if want.Logs == nil {
			want.Logs = []string{}
		}

		run := repo.runs[0]
		assert.Equal(t, tc.status, run.Status, fmt.Sprintf("%s: unexpected rule run status", tc.desc))
		assert.Equal(t, want, run.Details.Conditions[1], fmt.Sprintf("%s: unexpected condition result", tc.desc))
	}
}
