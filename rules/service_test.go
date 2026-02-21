// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package rules_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
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
	groupID    = "574106f7-030e-4881-8ab0-151195c29f94"
	otherGroup = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
)

func threshold(v float64) *float64 { return &v }

func newService() rules.Service {
	ths := authmock.NewThingsServiceClient(
		nil,
		map[string]things.Thing{
			token:   {ID: thingID, GroupID: groupID},
			thingID: {ID: thingID, GroupID: groupID},
		},
		map[string]things.Group{
			token: {ID: groupID},
		},
	)

	rulesRepo := mocks.NewRuleRepository()
	pubsub := mocks.NewPubSub()
	idp := uuid.NewMock()
	log := logger.NewMock()

	return rules.New(rulesRepo, ths, pubsub, idp, log)
}

func saveRules(t *testing.T, svc rules.Service, n int) []rules.Rule {
	t.Helper()

	var rs []rules.Rule
	for i := range n {
		rs = append(rs, rules.Rule{
			Name:        fmt.Sprintf("rule-%d", i),
			Description: fmt.Sprintf("desc-%d", i),
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

func assignRules(t *testing.T, svc rules.Service, thID string, ruleIDs ...string) {
	t.Helper()

	err := svc.AssignRules(context.Background(), token, thID, ruleIDs...)
	require.Nil(t, err)
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()

	b, err := json.Marshal(v)
	require.Nil(t, err)

	return b
}

func TestConsume(t *testing.T) {
	defaultConditions := []rules.Condition{
		{Field: "temperature", Comparator: ">", Threshold: threshold(25)},
	}

	cases := []struct {
		desc       string
		conditions []rules.Condition
		operator   string
		msg        any
		wantErr    bool
		err        error
	}{
		{
			desc:       "valid JSON message triggers rule",
			conditions: defaultConditions,
			operator:   rules.OperatorAND,
			msg: protomfx.Message{
				Publisher:   thingID,
				Payload:     mustMarshal(t, map[string]any{"temperature": float64(30)}),
				ContentType: "application/json",
			},
			err: nil,
		},
		{
			desc:       "JSON message below threshold does not trigger",
			conditions: defaultConditions,
			operator:   rules.OperatorAND,
			msg: protomfx.Message{
				Publisher:   thingID,
				Payload:     mustMarshal(t, map[string]any{"temperature": float64(20)}),
				ContentType: "application/json",
			},
			err: nil,
		},
		{
			desc:       "JSON array payload with partial match",
			conditions: defaultConditions,
			operator:   rules.OperatorAND,
			msg: protomfx.Message{
				Publisher: thingID,
				Payload: mustMarshal(t, []any{
					map[string]any{"temperature": float64(30)},
					map[string]any{"temperature": float64(10)},
				}),
				ContentType: "application/json",
			},
			err: nil,
		},
		{
			desc:       "SenML payload triggers rule",
			conditions: defaultConditions,
			operator:   rules.OperatorAND,
			msg: protomfx.Message{
				Publisher: thingID,
				Payload: mustMarshal(t, []any{
					map[string]any{"name": "temperature", "value": float64(30)},
				}),
				ContentType: "application/senml+json",
			},
			err: nil,
		},
		{
			// json.Unmarshal returns a stdlib error whose message varies by input,
			// so we only assert that an error is returned.
			desc:       "invalid JSON payload returns error",
			conditions: defaultConditions,
			operator:   rules.OperatorAND,
			msg: protomfx.Message{
				Publisher:   thingID,
				Payload:     []byte("not-json"),
				ContentType: "application/json",
			},
			wantErr: true,
		},
		{
			desc:    "non-Message type returns error",
			msg:     "not-a-message",
			wantErr: false,
			err:     errors.ErrMessage,
		},
		{
			desc:       "unknown publisher has no assigned rules",
			conditions: defaultConditions,
			operator:   rules.OperatorAND,
			msg: protomfx.Message{
				Publisher:   "2c8d1c97-6121-4c49-8a85-2baffd4e9e49",
				Payload:     mustMarshal(t, map[string]any{"temperature": float64(30)}),
				ContentType: "application/json",
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
				Publisher:   thingID,
				Payload:     mustMarshal(t, map[string]any{"temperature": float64(25), "humidity": float64(10)}),
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
				Publisher:   thingID,
				Payload:     mustMarshal(t, map[string]any{"temperature": float64(25), "humidity": float64(10)}),
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
				Publisher:   thingID,
				Payload:     mustMarshal(t, map[string]any{"temperature": float64(25), "humidity": float64(10)}),
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
				Publisher:   thingID,
				Payload:     mustMarshal(t, map[string]any{"temperature": float64(25), "humidity": float64(25)}),
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
				Publisher:   thingID,
				Payload:     mustMarshal(t, map[string]any{"temperature": float64(25)}),
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
				Publisher:   thingID,
				Payload:     mustMarshal(t, map[string]any{"temperature": float64(25)}),
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
				Publisher:   thingID,
				Payload:     mustMarshal(t, map[string]any{"temperature": float64(25)}),
				ContentType: "application/json",
			},
			err: nil,
		},
	}

	for _, tc := range cases {
		svc := newService()

		if len(tc.conditions) > 0 {
			saved, err := svc.CreateRules(context.Background(), token, groupID, rules.Rule{
				Name:       "test-rule",
				Conditions: tc.conditions,
				Operator:   tc.operator,
				Actions:    []rules.Action{{Type: rules.ActionTypeAlarm}},
			})
			require.Nil(t, err)
			assignRules(t, svc, thingID, saved[0].ID)
		}

		err := svc.Consume(tc.msg)
		if tc.wantErr {
			assert.NotNil(t, err, fmt.Sprintf("%s: expected an error, got nil", tc.desc))
		} else {
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
		}
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

	cases := []struct {
		desc    string
		token   string
		groupID string
		rules   []rules.Rule
		err     error
	}{
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
		pageMetadata apiutil.PageMetadata
		size         uint64
		err          error
	}{
		{
			desc:    "list rules by group",
			token:   token,
			groupID: groupID,
			pageMetadata: apiutil.PageMetadata{
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
			pageMetadata: apiutil.PageMetadata{
				Limit: 0,
			},
			size: uint64(n),
			err:  nil,
		},
		{
			desc:    "list last rule by group",
			token:   token,
			groupID: groupID,
			pageMetadata: apiutil.PageMetadata{
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
			pageMetadata: apiutil.PageMetadata{
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
			pageMetadata: apiutil.PageMetadata{
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
			pageMetadata: apiutil.PageMetadata{
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
	saved := saveRules(t, svc, n)

	var ruleIDs []string
	for _, r := range saved {
		ruleIDs = append(ruleIDs, r.ID)
	}
	assignRules(t, svc, thingID, ruleIDs...)

	cases := []struct {
		desc         string
		token        string
		thingID      string
		pageMetadata apiutil.PageMetadata
		size         uint64
		err          error
	}{
		{
			desc:    "list rules by thing",
			token:   token,
			thingID: thingID,
			pageMetadata: apiutil.PageMetadata{
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
			pageMetadata: apiutil.PageMetadata{
				Limit: 0,
			},
			size: uint64(n),
			err:  nil,
		},
		{
			desc:    "list last rule by thing",
			token:   token,
			thingID: thingID,
			pageMetadata: apiutil.PageMetadata{
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
			pageMetadata: apiutil.PageMetadata{
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
			pageMetadata: apiutil.PageMetadata{
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
			pageMetadata: apiutil.PageMetadata{
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
	assignRules(t, svc, thingID, ruleID)

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
		_, err := svc.ViewRule(context.Background(), tc.token, tc.ruleID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}

func TestUpdateRule(t *testing.T) {
	svc := newService()
	saved := saveRules(t, svc, 1)
	ruleID := saved[0].ID

	updated := rules.Rule{
		ID:          ruleID,
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
	n := 3
	saveRules(t, svc, n)

	page, err := svc.ListRulesByGroup(context.Background(), token, groupID, apiutil.PageMetadata{Limit: 20})
	require.Nil(t, err)
	require.Equal(t, n, len(page.Rules))

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

	page, err = svc.ListRulesByGroup(context.Background(), token, groupID, apiutil.PageMetadata{})
	require.Nil(t, err)
	assert.Equal(t, 0, len(page.Rules), "expected no rules after removal by group")
}

func TestAssignRules(t *testing.T) {
	svc := newService()
	saved := saveRules(t, svc, 2)
	ruleID := saved[0].ID


	cases := []struct {
		desc    string
		token   string
		thingID string
		ruleIDs []string
		err     error
	}{
		{
			desc:    "assign rules to thing",
			token:   token,
			thingID: thingID,
			ruleIDs: []string{ruleID},
			err:     nil,
		},
		{
			desc:    "assign rules with invalid token",
			token:   wrongValue,
			thingID: thingID,
			ruleIDs: []string{ruleID},
			err:     errors.ErrAuthentication,
		},
		{
			desc:    "assign non-existing rule to thing",
			token:   token,
			thingID: thingID,
			ruleIDs: []string{wrongValue},
			err:     dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.AssignRules(context.Background(), tc.token, tc.thingID, tc.ruleIDs...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}

func TestUnassignRules(t *testing.T) {
	svc := newService()
	saved := saveRules(t, svc, 1)
	ruleID := saved[0].ID
	assignRules(t, svc, thingID, ruleID)

	cases := []struct {
		desc    string
		token   string
		thingID string
		ruleIDs []string
		err     error
	}{
		{
			desc:    "unassign rules with invalid token",
			token:   wrongValue,
			thingID: thingID,
			ruleIDs: []string{ruleID},
			err:     errors.ErrAuthentication,
		},
		{
			desc:    "unassign rules from thing",
			token:   token,
			thingID: thingID,
			ruleIDs: []string{ruleID},
			err:     nil,
		},
		{
			desc:    "unassign non-existing rule from thing",
			token:   token,
			thingID: thingID,
			ruleIDs: []string{wrongValue},
			err:     dbutil.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.UnassignRules(context.Background(), tc.token, tc.thingID, tc.ruleIDs...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}
}

func TestUnassignRulesByThing(t *testing.T) {
	svc := newService()
	n := 3
	saved := saveRules(t, svc, n)

	var ruleIDs []string
	for _, r := range saved {
		ruleIDs = append(ruleIDs, r.ID)
	}
	assignRules(t, svc, thingID, ruleIDs...)

	page, err := svc.ListRulesByThing(context.Background(), token, thingID, apiutil.PageMetadata{Limit: 20})
	require.Nil(t, err)
	require.Equal(t, n, len(page.Rules))

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
		err := svc.UnassignRulesByThing(context.Background(), tc.thingID)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s", tc.desc, tc.err, err))
	}

	page, err = svc.ListRulesByThing(context.Background(), token, thingID, apiutil.PageMetadata{Limit: 20})
	require.Nil(t, err)
	assert.Equal(t, 0, len(page.Rules), "expected no rules after unassign by thing")
}
