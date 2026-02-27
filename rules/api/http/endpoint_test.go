// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	pkgmocks "github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/rules"
	httpapi "github.com/MainfluxLabs/mainflux/rules/api/http"
	rulesmocks "github.com/MainfluxLabs/mainflux/rules/mocks"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	token       = "admin@example.com"
	wrongValue  = "wrong-value"
	emptyValue  = ""
	contentType = "application/json"
	thingID     = "5384fb1c-d0ae-4cbe-be52-c54223150fe0"
	groupID     = "574106f7-030e-4881-8ab0-151195c29f94"
	ruleName    = "test-rule"
	prefixID    = "fe6b4e92-cc98-425e-b0aa-"
	prefixName  = "test-rule-"
)

var (
	threshold = 30.0
	condition = rules.Condition{Field: "temperature", Comparator: ">", Threshold: &threshold}
	action    = rules.Action{Type: rules.ActionTypeAlarm}
)

type ruleRes struct {
	ID          string            `json:"id"`
	GroupID     string            `json:"group_id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Conditions  []rules.Condition `json:"conditions"`
	Operator    string            `json:"operator"`
	Actions     []rules.Action    `json:"actions"`
}

type rulesPageRes struct {
	Total  uint64    `json:"total"`
	Offset uint64    `json:"offset"`
	Limit  uint64    `json:"limit"`
	Rules  []ruleRes `json:"rules"`
}

type thingIDsRes struct {
	ThingIDs []string `json:"thing_ids"`
}

func newService() rules.Service {
	ths := pkgmocks.NewThingsServiceClient(
		nil,
		map[string]things.Thing{
			token:   {ID: thingID, GroupID: groupID},
			thingID: {ID: thingID, GroupID: groupID},
		},
		map[string]things.Group{
			token: {ID: groupID},
		},
	)
	rulesRepo := rulesmocks.NewRuleRepository()
	pubsub := rulesmocks.NewPubSub()
	idp := uuid.NewMock()
	log := logger.NewMock()

	return rules.New(rulesRepo, ths, pubsub, idp, log)
}

func newHTTPServer(svc rules.Service) *httptest.Server {
	log := logger.NewMock()
	mux := httpapi.MakeHandler(mocktracer.New(), svc, log)
	return httptest.NewServer(mux)
}

func toJSON(data any) string {
	b, _ := json.Marshal(data)
	return string(b)
}

type testRequest struct {
	client      *http.Client
	method      string
	url         string
	contentType string
	token       string
	body        io.Reader
}

func (tr testRequest) make() (*http.Response, error) {
	req, err := http.NewRequest(tr.method, tr.url, tr.body)
	if err != nil {
		return nil, err
	}
	if tr.token != emptyValue {
		req.Header.Set("Authorization", apiutil.BearerPrefix+tr.token)
	}
	if tr.contentType != emptyValue {
		req.Header.Set("Content-Type", tr.contentType)
	}
	return tr.client.Do(req)
}

func saveRules(t *testing.T, svc rules.Service, n int) []rules.Rule {
	t.Helper()
	var saved []rules.Rule
	for i := range n {
		r := rules.Rule{
			ID:         fmt.Sprintf("%s%012d", prefixID, i+1),
			Name:       fmt.Sprintf("%s%012d", prefixName, i+1),
			Conditions: []rules.Condition{condition},
			Actions:    []rules.Action{action},
		}
		rs, err := svc.CreateRules(context.Background(), token, groupID, r)
		require.Nil(t, err, fmt.Sprintf("unexpected error saving rule %d: %s", i+1, err))
		saved = append(saved, rs...)
	}
	return saved
}

func TestCreateRules(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	makeRuleBody := func(operator string, conds ...any) string {
		r := map[string]any{
			"name":       ruleName,
			"conditions": conds,
			"actions":    []any{map[string]any{"type": rules.ActionTypeAlarm}},
		}
		if operator != "" {
			r["operator"] = operator
		}
		return toJSON(map[string]any{"rules": []any{r}})
	}

	condTemp := map[string]any{"field": "temperature", "comparator": ">", "threshold": 30.0}
	condHum := map[string]any{"field": "humidity", "comparator": "<", "threshold": 80.0}
	conditions := []any{condTemp, condHum}

	ruleBody := makeRuleBody("", condTemp)
	ruleWithANDOpBody := makeRuleBody(rules.OperatorAND, conditions...)
	ruleWithOROpBody := makeRuleBody(rules.OperatorOR, conditions...)
	ruleWithMissingOpBody := makeRuleBody("", conditions...)
	ruleWithInvalidOpBody := makeRuleBody("XOR", conditions...)

	multipleRulesBody := toJSON(map[string]any{
		"rules": []any{
			map[string]any{
				"name":       "rule-1",
				"conditions": []any{condTemp},
				"actions":    []any{map[string]any{"type": rules.ActionTypeAlarm}},
			},
			map[string]any{
				"name":       "rule-2",
				"conditions": []any{condHum},
				"actions":    []any{map[string]any{"type": rules.ActionTypeAlarm}},
			},
		},
	})

	cases := []struct {
		desc        string
		auth        string
		groupID     string
		contentType string
		body        string
		status      int
		size        int
	}{
		{
			desc:        "create valid rule",
			auth:        token,
			groupID:     groupID,
			contentType: contentType,
			body:        ruleBody,
			status:      http.StatusCreated,
			size:        1,
		},
		{
			desc:        "create multiple rules",
			auth:        token,
			groupID:     groupID,
			contentType: contentType,
			body:        multipleRulesBody,
			status:      http.StatusCreated,
			size:        2,
		},
		{
			desc:        "create rule with multiple conditions and AND operator",
			auth:        token,
			groupID:     groupID,
			contentType: contentType,
			body:        ruleWithANDOpBody,
			status:      http.StatusCreated,
			size:        1,
		},
		{
			desc:        "create rule with multiple conditions and OR operator",
			auth:        token,
			groupID:     groupID,
			contentType: contentType,
			body:        ruleWithOROpBody,
			status:      http.StatusCreated,
			size:        1,
		},
		{
			desc:        "create rule with multiple conditions and missing operator",
			auth:        token,
			groupID:     groupID,
			contentType: contentType,
			body:        ruleWithMissingOpBody,
			status:      http.StatusBadRequest,
			size:        0,
		},
		{
			desc:        "create rule with multiple conditions and invalid operator",
			auth:        token,
			groupID:     groupID,
			contentType: contentType,
			body:        ruleWithInvalidOpBody,
			status:      http.StatusBadRequest,
			size:        0,
		},
		{
			desc:        "create rule with empty token",
			auth:        emptyValue,
			groupID:     groupID,
			contentType: contentType,
			body:        ruleBody,
			status:      http.StatusUnauthorized,
			size:        0,
		},
		{
			desc:        "create rule with wrong token",
			auth:        wrongValue,
			groupID:     groupID,
			contentType: contentType,
			body:        ruleBody,
			status:      http.StatusUnauthorized,
			size:        0,
		},
		{
			desc:        "create rule with wrong group ID",
			auth:        token,
			groupID:     wrongValue,
			contentType: contentType,
			body:        ruleBody,
			status:      http.StatusForbidden,
			size:        0,
		},
		{
			desc:        "create rule without content type",
			auth:        token,
			groupID:     groupID,
			contentType: emptyValue,
			body:        ruleBody,
			status:      http.StatusUnsupportedMediaType,
			size:        0,
		},
		{
			desc:        "create rule with malformed JSON",
			auth:        token,
			groupID:     groupID,
			contentType: contentType,
			body:        "}{",
			status:      http.StatusBadRequest,
			size:        0,
		},
		{
			desc:        "create rule with empty body",
			auth:        token,
			groupID:     groupID,
			contentType: contentType,
			body:        emptyValue,
			status:      http.StatusBadRequest,
			size:        0,
		},
		{
			desc:        "create rule with empty rules array",
			auth:        token,
			groupID:     groupID,
			contentType: contentType,
			body:        toJSON(map[string]any{"rules": []any{}}),
			status:      http.StatusBadRequest,
			size:        0,
		},
		{
			desc:        "create rule with missing conditions",
			auth:        token,
			groupID:     groupID,
			contentType: contentType,
			body: toJSON(map[string]any{
				"rules": []any{
					map[string]any{
						"name":    ruleName,
						"actions": []any{map[string]any{"type": rules.ActionTypeAlarm}},
					},
				},
			}),
			status: http.StatusBadRequest,
			size:   0,
		},
		{
			desc:        "create rule with missing actions",
			auth:        token,
			groupID:     groupID,
			contentType: contentType,
			body: toJSON(map[string]any{
				"rules": []any{
					map[string]any{
						"name":       ruleName,
						"conditions": []any{condTemp},
					},
				},
			}),
			status: http.StatusBadRequest,
			size:   0,
		},
		{
			desc:        "create rule with invalid action type",
			auth:        token,
			groupID:     groupID,
			contentType: contentType,
			body: toJSON(map[string]any{
				"rules": []any{
					map[string]any{
						"name":       ruleName,
						"conditions": []any{condTemp},
						"actions":    []any{map[string]any{"type": "unknown"}},
					},
				},
			}),
			status: http.StatusBadRequest,
			size:   0,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/groups/%s/rules", ts.URL, tc.groupID),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.body),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s\n", tc.desc, err))

		var body struct {
			Rules []ruleRes `json:"rules"`
		}
		json.NewDecoder(res.Body).Decode(&body)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d\n", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.size, len(body.Rules), fmt.Sprintf("%s: expected %d rules got %d\n", tc.desc, tc.size, len(body.Rules)))
	}
}

func TestViewRule(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	saved := saveRules(t, svc, 1)
	ruleID := saved[0].ID

	cases := []struct {
		desc   string
		auth   string
		id     string
		status int
	}{
		{
			desc:   "view existing rule",
			auth:   token,
			id:     ruleID,
			status: http.StatusOK,
		},
		{
			desc:   "view rule with empty token",
			auth:   emptyValue,
			id:     ruleID,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "view rule with wrong token",
			auth:   wrongValue,
			id:     ruleID,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "view rule with empty ID",
			auth:   token,
			id:     emptyValue,
			status: http.StatusBadRequest,
		},
		{
			desc:   "view non-existing rule",
			auth:   token,
			id:     wrongValue,
			status: http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/rules/%s", ts.URL, tc.id),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s\n", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d\n", tc.desc, tc.status, res.StatusCode))
	}
}

func TestListRulesByGroup(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	n := 10
	saved := saveRules(t, svc, n)
	_ = saved

	cases := []struct {
		desc   string
		auth   string
		url    string
		status int
		size   int
	}{
		{
			desc:   "list rules by group",
			auth:   token,
			url:    fmt.Sprintf("%s/groups/%s/rules?limit=%d&offset=0", ts.URL, groupID, n),
			status: http.StatusOK,
			size:   n,
		},
		{
			desc:   "list rules by group with default limit",
			auth:   token,
			url:    fmt.Sprintf("%s/groups/%s/rules", ts.URL, groupID),
			status: http.StatusOK,
			size:   n,
		},
		{
			desc:   "list rules by group with limit",
			auth:   token,
			url:    fmt.Sprintf("%s/groups/%s/rules?limit=5&offset=0", ts.URL, groupID),
			status: http.StatusOK,
			size:   5,
		},
		{
			desc:   "list rules by group with offset",
			auth:   token,
			url:    fmt.Sprintf("%s/groups/%s/rules?limit=%d&offset=%d", ts.URL, groupID, n, n-1),
			status: http.StatusOK,
			size:   1,
		},
		{
			desc:   "list rules by group with limit exceeding max",
			auth:   token,
			url:    fmt.Sprintf("%s/groups/%s/rules?limit=201", ts.URL, groupID),
			status: http.StatusBadRequest,
			size:   0,
		},
		{
			desc:   "list rules by group with invalid limit",
			auth:   token,
			url:    fmt.Sprintf("%s/groups/%s/rules?limit=invalid", ts.URL, groupID),
			status: http.StatusBadRequest,
			size:   0,
		},
		{
			desc:   "list rules by group with invalid offset",
			auth:   token,
			url:    fmt.Sprintf("%s/groups/%s/rules?offset=invalid", ts.URL, groupID),
			status: http.StatusBadRequest,
			size:   0,
		},
		{
			desc:   "list rules by group with wrong group ID",
			auth:   token,
			url:    fmt.Sprintf("%s/groups/%s/rules?limit=%d", ts.URL, wrongValue, n),
			status: http.StatusForbidden,
			size:   0,
		},
		{
			desc:   "list rules by group with empty token",
			auth:   emptyValue,
			url:    fmt.Sprintf("%s/groups/%s/rules?limit=%d", ts.URL, groupID, n),
			status: http.StatusUnauthorized,
			size:   0,
		},
		{
			desc:   "list rules by group with wrong token",
			auth:   wrongValue,
			url:    fmt.Sprintf("%s/groups/%s/rules?limit=%d", ts.URL, groupID, n),
			status: http.StatusUnauthorized,
			size:   0,
		},
		{
			desc:   "list rules by group sorted by name ascending",
			auth:   token,
			url:    fmt.Sprintf("%s/groups/%s/rules?order=name&dir=asc&limit=%d", ts.URL, groupID, 5),
			status: http.StatusOK,
			size:   5,
		},
		{
			desc:   "list rules by group with invalid order",
			auth:   token,
			url:    fmt.Sprintf("%s/groups/%s/rules?order=invalid&dir=asc&limit=%d", ts.URL, groupID, n),
			status: http.StatusBadRequest,
			size:   0,
		},
		{
			desc:   "list rules by group with invalid direction",
			auth:   token,
			url:    fmt.Sprintf("%s/groups/%s/rules?order=name&dir=invalid&limit=%d", ts.URL, groupID, n),
			status: http.StatusBadRequest,
			size:   0,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    tc.url,
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s\n", tc.desc, err))

		var page rulesPageRes
		json.NewDecoder(res.Body).Decode(&page)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d\n", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.size, len(page.Rules), fmt.Sprintf("%s: expected size %d got %d\n", tc.desc, tc.size, len(page.Rules)))
	}
}

func TestListRulesByThing(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	n := 5
	saved := saveRules(t, svc, n)

	var ruleIDs []string
	for _, r := range saved {
		ruleIDs = append(ruleIDs, r.ID)
	}
	err := svc.AssignRules(context.Background(), token, thingID, ruleIDs...)
	require.Nil(t, err, fmt.Sprintf("unexpected error assigning rules: %s", err))

	cases := []struct {
		desc   string
		auth   string
		url    string
		status int
		size   int
	}{
		{
			desc:   "list rules by thing",
			auth:   token,
			url:    fmt.Sprintf("%s/things/%s/rules?limit=%d&offset=0", ts.URL, thingID, n),
			status: http.StatusOK,
			size:   n,
		},
		{
			desc:   "list rules by thing with limit",
			auth:   token,
			url:    fmt.Sprintf("%s/things/%s/rules?limit=3&offset=0", ts.URL, thingID),
			status: http.StatusOK,
			size:   3,
		},
		{
			desc:   "list rules by thing with wrong thing ID",
			auth:   token,
			url:    fmt.Sprintf("%s/things/%s/rules?limit=%d", ts.URL, wrongValue, n),
			status: http.StatusForbidden,
			size:   0,
		},
		{
			desc:   "list rules by thing with empty token",
			auth:   emptyValue,
			url:    fmt.Sprintf("%s/things/%s/rules?limit=%d", ts.URL, thingID, n),
			status: http.StatusUnauthorized,
			size:   0,
		},
		{
			desc:   "list rules by thing with limit exceeding max",
			auth:   token,
			url:    fmt.Sprintf("%s/things/%s/rules?limit=201", ts.URL, thingID),
			status: http.StatusBadRequest,
			size:   0,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    tc.url,
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s\n", tc.desc, err))

		var page rulesPageRes
		json.NewDecoder(res.Body).Decode(&page)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d\n", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.size, len(page.Rules), fmt.Sprintf("%s: expected size %d got %d\n", tc.desc, tc.size, len(page.Rules)))
	}
}

func TestListThingIDsByRule(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	saved := saveRules(t, svc, 1)
	ruleID := saved[0].ID

	err := svc.AssignRules(context.Background(), token, thingID, ruleID)
	require.Nil(t, err, fmt.Sprintf("unexpected error assigning rule: %s", err))

	cases := []struct {
		desc   string
		auth   string
		id     string
		status int
		size   int
	}{
		{
			desc:   "list thing IDs by rule",
			auth:   token,
			id:     ruleID,
			status: http.StatusOK,
			size:   1,
		},
		{
			desc:   "list thing IDs by non-existing rule",
			auth:   token,
			id:     wrongValue,
			status: http.StatusNotFound,
			size:   0,
		},
		{
			desc:   "list thing IDs by rule with empty token",
			auth:   emptyValue,
			id:     ruleID,
			status: http.StatusUnauthorized,
			size:   0,
		},
		{
			desc:   "list thing IDs by rule with empty rule ID",
			auth:   token,
			id:     emptyValue,
			status: http.StatusBadRequest,
			size:   0,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/rules/%s/things", ts.URL, tc.id),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s\n", tc.desc, err))

		var body thingIDsRes
		json.NewDecoder(res.Body).Decode(&body)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d\n", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.size, len(body.ThingIDs), fmt.Sprintf("%s: expected %d thing IDs got %d\n", tc.desc, tc.size, len(body.ThingIDs)))
	}
}

func TestUpdateRule(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	saved := saveRules(t, svc, 1)
	ruleID := saved[0].ID

	updatedBody := toJSON(map[string]any{
		"name":       "updated-rule",
		"conditions": []any{map[string]any{"field": "temperature", "comparator": ">", "threshold": 35.0}},
		"actions":    []any{map[string]any{"type": rules.ActionTypeAlarm}},
	})

	cases := []struct {
		desc        string
		auth        string
		id          string
		contentType string
		body        string
		status      int
	}{
		{
			desc:        "update existing rule",
			auth:        token,
			id:          ruleID,
			contentType: contentType,
			body:        updatedBody,
			status:      http.StatusOK,
		},
		{
			desc:        "update rule with empty token",
			auth:        emptyValue,
			id:          ruleID,
			contentType: contentType,
			body:        updatedBody,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update rule with wrong token",
			auth:        wrongValue,
			id:          ruleID,
			contentType: contentType,
			body:        updatedBody,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update non-existing rule",
			auth:        token,
			id:          wrongValue,
			contentType: contentType,
			body:        updatedBody,
			status:      http.StatusNotFound,
		},
		{
			desc:        "update rule without content type",
			auth:        token,
			id:          ruleID,
			contentType: emptyValue,
			body:        updatedBody,
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "update rule with malformed JSON",
			auth:        token,
			id:          ruleID,
			contentType: contentType,
			body:        "}{",
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update rule with missing conditions",
			auth:        token,
			id:          ruleID,
			contentType: contentType,
			body: toJSON(map[string]any{
				"name":    "updated-rule",
				"actions": []any{map[string]any{"type": rules.ActionTypeAlarm}},
			}),
			status: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPut,
			url:         fmt.Sprintf("%s/rules/%s", ts.URL, tc.id),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.body),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s\n", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d\n", tc.desc, tc.status, res.StatusCode))
	}
}

func TestRemoveRules(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	saved := saveRules(t, svc, 1)
	ruleID := saved[0].ID

	cases := []struct {
		desc        string
		auth        string
		contentType string
		ids         []string
		status      int
	}{
		{
			desc:        "remove rules with empty token",
			auth:        emptyValue,
			contentType: contentType,
			ids:         []string{ruleID},
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "remove existing rules",
			auth:        token,
			contentType: contentType,
			ids:         []string{ruleID},
			status:      http.StatusNoContent,
		},
		{
			desc:        "remove non-existing rules",
			auth:        token,
			contentType: contentType,
			ids:         []string{wrongValue},
			status:      http.StatusNotFound,
		},
		{
			desc:        "remove rules with empty list",
			auth:        token,
			contentType: contentType,
			ids:         []string{},
			status:      http.StatusBadRequest,
		},
		{
			desc:        "remove rules without content type",
			auth:        token,
			contentType: emptyValue,
			ids:         []string{ruleID},
			status:      http.StatusUnsupportedMediaType,
		},
	}

	for _, tc := range cases {
		body := toJSON(struct {
			RuleIDs []string `json:"rule_ids"`
		}{tc.ids})

		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/rules", ts.URL),
			token:       tc.auth,
			contentType: tc.contentType,
			body:        strings.NewReader(body),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s\n", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d\n", tc.desc, tc.status, res.StatusCode))
	}
}

func TestAssignRules(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	saved := saveRules(t, svc, 2)
	ruleID1 := saved[0].ID
	ruleID2 := saved[1].ID

	cases := []struct {
		desc        string
		auth        string
		thingID     string
		contentType string
		ids         []string
		status      int
	}{
		{
			desc:        "assign rules to thing",
			auth:        token,
			thingID:     thingID,
			contentType: contentType,
			ids:         []string{ruleID1, ruleID2},
			status:      http.StatusOK,
		},
		{
			desc:        "assign rules with empty token",
			auth:        emptyValue,
			thingID:     thingID,
			contentType: contentType,
			ids:         []string{ruleID1},
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "assign rules with wrong token",
			auth:        wrongValue,
			thingID:     thingID,
			contentType: contentType,
			ids:         []string{ruleID1},
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "assign rules to wrong thing ID",
			auth:        token,
			thingID:     wrongValue,
			contentType: contentType,
			ids:         []string{ruleID1},
			status:      http.StatusForbidden,
		},
		{
			desc:        "assign rules without content type",
			auth:        token,
			thingID:     thingID,
			contentType: emptyValue,
			ids:         []string{ruleID1},
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "assign rules with empty list",
			auth:        token,
			thingID:     thingID,
			contentType: contentType,
			ids:         []string{},
			status:      http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		body := toJSON(struct {
			RuleIDs []string `json:"rule_ids"`
		}{tc.ids})

		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/things/%s/rules", ts.URL, tc.thingID),
			token:       tc.auth,
			contentType: tc.contentType,
			body:        strings.NewReader(body),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s\n", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d\n", tc.desc, tc.status, res.StatusCode))
	}
}

func TestUnassignRules(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	saved := saveRules(t, svc, 2)
	ruleID1 := saved[0].ID
	ruleID2 := saved[1].ID

	err := svc.AssignRules(context.Background(), token, thingID, ruleID1, ruleID2)
	require.Nil(t, err, fmt.Sprintf("unexpected error assigning rules: %s", err))

	cases := []struct {
		desc        string
		auth        string
		thingID     string
		contentType string
		ids         []string
		status      int
	}{
		{
			desc:        "unassign rules from thing",
			auth:        token,
			thingID:     thingID,
			contentType: contentType,
			ids:         []string{ruleID1, ruleID2},
			status:      http.StatusOK,
		},
		{
			desc:        "unassign rules with empty token",
			auth:        emptyValue,
			thingID:     thingID,
			contentType: contentType,
			ids:         []string{ruleID1},
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "unassign rules from wrong thing ID",
			auth:        token,
			thingID:     wrongValue,
			contentType: contentType,
			ids:         []string{ruleID1},
			status:      http.StatusForbidden,
		},
		{
			desc:        "unassign rules without content type",
			auth:        token,
			thingID:     thingID,
			contentType: emptyValue,
			ids:         []string{ruleID1},
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "unassign rules with empty list",
			auth:        token,
			thingID:     thingID,
			contentType: contentType,
			ids:         []string{},
			status:      http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		body := toJSON(struct {
			RuleIDs []string `json:"rule_ids"`
		}{tc.ids})

		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/things/%s/rules", ts.URL, tc.thingID),
			token:       tc.auth,
			contentType: tc.contentType,
			body:        strings.NewReader(body),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s\n", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d\n", tc.desc, tc.status, res.StatusCode))
	}
}
