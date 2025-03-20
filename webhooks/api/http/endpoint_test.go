package http_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/MainfluxLabs/mainflux/webhooks"
	httpapi "github.com/MainfluxLabs/mainflux/webhooks/api/http"
	whmocks "github.com/MainfluxLabs/mainflux/webhooks/mocks"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	token       = "admin@example.com"
	wrongValue  = "wrong-value"
	invalidUrl  = "invalid-url"
	contentType = "application/json"
	emptyValue  = ""
	groupID     = "50e6b371-60ff-45cf-bb52-8200e7cde536"
	thingID     = "5384fb1c-d0ae-4cbe-be52-c54223150fe0"
	prefixID    = "fe6b4e92-cc98-425e-b0aa-"
	prefixName  = "test-webhook-"
	nameKey     = "name"
	ascKey      = "asc"
	descKey     = "desc"
)

var (
	headers       = map[string]string{"Content-Type:": "application/json"}
	webhook       = webhooks.Webhook{ThingID: thingID, GroupID: groupID, Name: "test-webhook", Url: "https://test.webhook.com", Headers: headers, Metadata: map[string]interface{}{"test": "data"}}
	invalidIDRes  = toJSON(apiutil.ErrorRes{Err: apiutil.ErrMissingWebhookID.Error()})
	missingTokRes = toJSON(apiutil.ErrorRes{Err: apiutil.ErrBearerToken.Error()})
)

func newHTTPServer(svc webhooks.Service) *httptest.Server {
	logger := logger.NewMock()
	mux := httpapi.MakeHandler(mocktracer.New(), svc, logger)
	return httptest.NewServer(mux)
}

func toJSON(data interface{}) string {
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

func newService() webhooks.Service {
	ths := mocks.NewThingsServiceClient(nil, map[string]things.Thing{thingID: {ID: thingID, GroupID: groupID}, token: {ID: thingID, GroupID: groupID}}, map[string]things.Group{token: {ID: groupID}})
	webhookRepo := whmocks.NewWebhookRepository()
	forwarder := whmocks.NewForwarder()
	idProvider := uuid.NewMock()

	return webhooks.New(ths, webhookRepo, forwarder, idProvider)
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

	if tr.token != "" {
		req.Header.Set("Authorization", apiutil.BearerPrefix+tr.token)
	}

	if tr.contentType != "" {
		req.Header.Set("Content-Type", tr.contentType)
	}

	return tr.client.Do(req)
}

func TestCreateWebhooks(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	validData := `[{"name":"value","url":"https://api.example.com","headers":{"Content-Type":"application/json"}}]`
	invalidName := fmt.Sprintf(`[{"name":"%s","url":"https://api.example.com","headers":{"Content-Type":"application/json"}}]`, emptyValue)
	invalidUrl := fmt.Sprintf(`[{"name":"value","url":"%s","headers":{"Content-Type":"application/json"}}]`, invalidUrl)

	cases := []struct {
		desc        string
		data        string
		thingID     string
		contentType string
		auth        string
		status      int
		response    string
	}{
		{
			desc:        "create valid webhooks",
			data:        validData,
			thingID:     thingID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusCreated,
			response:    emptyValue,
		},
		{
			desc:        "create webhooks with empty request",
			data:        emptyValue,
			thingID:     thingID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    emptyValue,
		},
		{
			desc:        "create webhooks with invalid request format",
			data:        "}{",
			thingID:     thingID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    emptyValue,
		},
		{
			desc:        "create webhooks with invalid thing id",
			data:        validData,
			thingID:     wrongValue,
			contentType: contentType,
			auth:        token,
			status:      http.StatusForbidden,
			response:    emptyValue,
		},
		{
			desc:        "create webhooks with empty thing id",
			data:        validData,
			thingID:     emptyValue,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    emptyValue,
		},
		{
			desc:        "create webhooks with invalid name",
			data:        invalidName,
			thingID:     thingID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    emptyValue,
		},
		{
			desc:        "create webhooks with invalid url",
			data:        invalidUrl,
			thingID:     thingID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    emptyValue,
		},
		{
			desc:        "create webhooks with empty JSON array",
			data:        "[]",
			thingID:     thingID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    emptyValue,
		},
		{
			desc:        "create webhook with wrong auth token",
			data:        validData,
			thingID:     thingID,
			contentType: contentType,
			auth:        wrongValue,
			status:      http.StatusUnauthorized,
			response:    emptyValue,
		},
		{
			desc:        "create webhook with empty auth token",
			data:        validData,
			thingID:     thingID,
			contentType: contentType,
			auth:        emptyValue,
			status:      http.StatusUnauthorized,
			response:    emptyValue,
		},
		{
			desc:        "create webhook without content type",
			data:        validData,
			thingID:     thingID,
			contentType: emptyValue,
			auth:        token,
			status:      http.StatusUnsupportedMediaType,
			response:    emptyValue,
		},
	}
	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/things/%s/webhooks", ts.URL, tc.thingID),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.data),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		location := res.Header.Get("Location")
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.response, location, fmt.Sprintf("%s: expected response %s got %s", tc.desc, tc.response, location))
	}
}

type webhookRes struct {
	ID         string                 `json:"id"`
	GroupID    string                 `json:"group_id"`
	ThingID    string                 `json:"thing_id"`
	Name       string                 `json:"name"`
	Url        string                 `json:"url"`
	ResHeaders map[string]string      `json:"headers"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

type webhooksPageRes struct {
	Webhooks []webhookRes `json:"webhooks"`
	Total    uint64       `json:"total"`
	Offset   uint64       `json:"offset"`
	Limit    uint64       `json:"limit"`
}

func TestListWebhooksByGroup(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	var data []webhookRes
	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("%s%012d", prefixID, i+1)
		name := fmt.Sprintf("%s%012d", prefixName, i+1)
		webhook1 := webhook
		webhook1.ID = id
		webhook1.Name = name

		whs, err := svc.CreateWebhooks(context.Background(), token, webhook1)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		w := whs[0]
		whRes := webhookRes{
			ID:         w.ID,
			GroupID:    w.GroupID,
			ThingID:    w.ThingID,
			Name:       w.Name,
			Url:        w.Url,
			ResHeaders: w.Headers,
			Metadata:   w.Metadata,
		}
		data = append(data, whRes)
	}

	cases := []struct {
		desc   string
		auth   string
		status int
		url    string
		res    []webhookRes
	}{
		{
			desc:   "get a list of webhooks by group",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/groups/%s/webhooks?offset=%d&limit=%d", ts.URL, groupID, 0, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of webhooks by group with no limit",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/groups/%s/webhooks?limit=%d", ts.URL, groupID, -1),
			res:    data,
		},
		{
			desc:   "get a list of webhooks by group with negative offset",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/groups/%s/webhooks?offset=%d&limit=%d", ts.URL, groupID, -2, 2),
			res:    nil,
		},
		{
			desc:   "get a list of webhooks by group with negative limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/groups/%s/webhooks?offset=%d&limit=%d", ts.URL, groupID, 2, -3),
			res:    nil,
		},
		{
			desc:   "get a list of webhooks by group with zero limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/groups/%s/webhooks?offset=%d&limit=%d", ts.URL, groupID, 2, 0),
			res:    nil,
		},
		{
			desc:   "get a list of webhooks by group with invalid token",
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s/groups/%s/webhooks?offset=%d&limit=%d", ts.URL, groupID, 0, 1),
			res:    nil,
		},
		{
			desc:   "get a list of webhooks by group with empty token",
			auth:   emptyValue,
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s/groups/%s/webhooks?offset=%d&limit=%d", ts.URL, groupID, 0, 1),
			res:    []webhookRes{},
		},
		{
			desc:   "get a list of webhooks by group with wrong group id",
			auth:   token,
			status: http.StatusForbidden,
			url:    fmt.Sprintf("%s/groups/%s/webhooks", ts.URL, wrongValue),
			res:    []webhookRes{},
		},
		{
			desc:   "get a list of webhooks by group without offset",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/groups/%s/webhooks?limit=%d", ts.URL, groupID, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of webhooks by group without limit",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/groups/%s/webhooks?offset=%d", ts.URL, groupID, 1),
			res:    data[1:10],
		},
		{
			desc:   "get a list of webhooks by group with redundant query params",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/groups/%s/webhooks?offset=%d&limit=%d&value=something", ts.URL, groupID, 0, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of webhooks by group with limit greater than max",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/groups/%s/webhooks?offset=%d&limit=%d", ts.URL, groupID, 0, 101),
			res:    nil,
		},
		{
			desc:   "get a list of webhooks by group with default URL",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/groups/%s/webhooks", ts.URL, groupID),
			res:    data[0:10],
		},
		{
			desc:   "get a list of webhooks by group with invalid number of params",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/groups/%s/webhooks%s", ts.URL, groupID, "?offset=4&limit=4&limit=5&offset=5"),
			res:    nil,
		},
		{
			desc:   "get a list of webhooks by group with invalid offset",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/groups/%s/webhooks%s", ts.URL, groupID, "?offset=e&limit=5"),
			res:    nil,
		},
		{
			desc:   "get a list of webhooks by group with invalid limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/groups/%s/webhooks%s", ts.URL, groupID, "?offset=5&limit=e"),
			res:    nil,
		},
		{
			desc:   "get a list of webhooks by group sorted by name ascendant",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/groups/%s/webhooks?offset=%d&limit=%d&order=%s&dir=%s", ts.URL, groupID, 0, 5, nameKey, ascKey),
			res:    data[0:5],
		},
		{
			desc:   "get a list of webhooks by group sorted by name descendent",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/groups/%s/webhooks?offset=%d&limit=%d&order=%s&dir=%s", ts.URL, groupID, 0, 5, nameKey, descKey),
			res:    data[0:5],
		},
		{
			desc:   "get a list of webhooks by group sorted with invalid order",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/groups/%s/webhooks?offset=%d&limit=%d&order=%s&dir=%s", ts.URL, groupID, 0, 5, wrongValue, ascKey),
			res:    nil,
		},
		{
			desc:   "get a list of webhooks by group sorted by name with invalid direction",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/groups/%s/webhooks?offset=%d&limit=%d&order=%s&dir=%s", ts.URL, groupID, 0, 5, nameKey, wrongValue),
			res:    nil,
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
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		var data webhooksPageRes
		json.NewDecoder(res.Body).Decode(&data)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res, data.Webhooks, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, data.Webhooks))

	}
}

func TestUpdateWebhook(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	data := toJSON(webhook)

	whs, err := svc.CreateWebhooks(context.Background(), token, webhook)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	wh1 := whs[0]

	wh2 := webhook
	wh2.Name = emptyValue
	invalidData := toJSON(wh2)

	cases := []struct {
		desc        string
		req         string
		id          string
		contentType string
		auth        string
		status      int
	}{
		{
			desc:        "update existing webhook",
			req:         data,
			id:          wh1.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusOK,
		},
		{
			desc:        "update webhook with empty JSON request",
			req:         "{}",
			id:          wh1.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update non-existent webhook",
			req:         data,
			id:          strconv.FormatUint(0, 10),
			contentType: contentType,
			auth:        token,
			status:      http.StatusNotFound,
		},
		{
			desc:        "update webhook with invalid id",
			req:         data,
			id:          "invalid",
			contentType: contentType,
			auth:        token,
			status:      http.StatusNotFound,
		},
		{
			desc:        "update webhook with empty user token",
			req:         data,
			id:          wh1.ID,
			contentType: contentType,
			auth:        emptyValue,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update webhook with invalid data format",
			req:         "{",
			id:          wh1.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update webhook with empty request",
			req:         "",
			id:          wh1.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update webhook without content type",
			req:         data,
			id:          wh1.ID,
			contentType: "",
			auth:        token,
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "update webhook with invalid name",
			req:         invalidData,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPut,
			url:         fmt.Sprintf("%s/webhooks/%s", ts.URL, tc.id),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestViewWebhook(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	whs, err := svc.CreateWebhooks(context.Background(), token, webhook)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	wh := whs[0]

	data := toJSON(webhookRes{
		ID:         wh.ID,
		ThingID:    wh.ThingID,
		GroupID:    wh.GroupID,
		Name:       wh.Name,
		Url:        wh.Url,
		ResHeaders: wh.Headers,
		Metadata:   wh.Metadata,
	})

	cases := []struct {
		desc   string
		id     string
		auth   string
		status int
		res    string
	}{
		{
			desc:   "view existing webhook",
			id:     wh.ID,
			auth:   token,
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "view webhook with empty token",
			id:     wh.ID,
			auth:   emptyValue,
			status: http.StatusUnauthorized,
			res:    missingTokRes,
		},
		{
			desc:   "view webhook with invalid id",
			id:     emptyValue,
			auth:   token,
			status: http.StatusBadRequest,
			res:    invalidIDRes,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/webhooks/%s", ts.URL, tc.id),
			token:  tc.auth,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		body, err := ioutil.ReadAll(res.Body)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		data := strings.Trim(string(body), "\n")
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.res, data, fmt.Sprintf("%s: expected body %s got %s", tc.desc, tc.res, data))
	}
}

func TestRemoveWebhooks(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	webhook2 := webhook
	webhook2.Name = "Test2"

	whs := []webhooks.Webhook{webhook, webhook2}
	grWhs, err := svc.CreateWebhooks(context.Background(), token, whs...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	var webhookIDs []string
	for _, wh := range grWhs {
		webhookIDs = append(webhookIDs, wh.ID)
	}

	cases := []struct {
		desc        string
		data        []string
		auth        string
		contentType string
		status      int
	}{
		{
			desc:        "remove existing webhooks",
			data:        webhookIDs,
			auth:        token,
			contentType: contentType,
			status:      http.StatusNoContent,
		},
		{
			desc:        "remove non-existent webhooks",
			data:        []string{wrongValue},
			auth:        token,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "remove webhooks with empty token",
			data:        webhookIDs,
			auth:        emptyValue,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "remove webhooks with invalid content type",
			data:        webhookIDs,
			auth:        token,
			contentType: wrongValue,
			status:      http.StatusUnsupportedMediaType,
		},
	}

	for _, tc := range cases {
		data := struct {
			WebhookIDs []string `json:"webhook_ids"`
		}{
			tc.data,
		}

		body := toJSON(data)

		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/webhooks", ts.URL),
			token:       tc.auth,
			contentType: tc.contentType,
			body:        strings.NewReader(body),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}
