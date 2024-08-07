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
)

var (
	headers       = map[string]string{"Content-Type:": "application/json"}
	webhook       = webhooks.Webhook{GroupID: groupID, Name: "test-webhook", Url: "https://test.webhook.com", Headers: headers}
	invalidIDRes  = toJSON(apiutil.ErrorRes{Err: apiutil.ErrInvalidIDFormat.Error()})
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
	things := mocks.NewThingsServiceClient(nil, map[string]string{token: groupID}, nil)
	webhookRepo := whmocks.NewWebhookRepository()
	forwarder := whmocks.NewForwarder()
	idProvider := uuid.NewMock()

	return webhooks.New(things, webhookRepo, forwarder, idProvider)
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
		groupID     string
		contentType string
		auth        string
		status      int
		response    string
	}{
		{
			desc:        "create valid webhooks",
			data:        validData,
			groupID:     groupID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusCreated,
			response:    emptyValue,
		},
		{
			desc:        "create webhooks with empty request",
			data:        emptyValue,
			groupID:     groupID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    emptyValue,
		},
		{
			desc:        "create webhooks with invalid request format",
			data:        "}{",
			groupID:     groupID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    emptyValue,
		},
		{
			desc:        "create webhooks with invalid group id",
			data:        validData,
			groupID:     wrongValue,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    emptyValue,
		},
		{
			desc:        "create webhooks with invalid name",
			data:        invalidName,
			groupID:     groupID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    emptyValue,
		},
		{
			desc:        "create webhooks with invalid url",
			data:        invalidUrl,
			groupID:     groupID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    emptyValue,
		},
		{
			desc:        "create webhooks with empty JSON array",
			data:        "[]",
			groupID:     groupID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    emptyValue,
		},
		{
			desc:        "create webhook with wrong auth token",
			data:        validData,
			groupID:     groupID,
			contentType: contentType,
			auth:        wrongValue,
			status:      http.StatusForbidden,
			response:    emptyValue,
		},
		{
			desc:        "create webhook with empty auth token",
			data:        validData,
			groupID:     groupID,
			contentType: contentType,
			auth:        emptyValue,
			status:      http.StatusUnauthorized,
			response:    emptyValue,
		},
		{
			desc:        "create webhook without content type",
			data:        validData,
			groupID:     groupID,
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
			url:         fmt.Sprintf("%s/groups/%s/webhooks", ts.URL, tc.groupID),
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
	ID         string            `json:"id"`
	GroupID    string            `json:"group_id"`
	Name       string            `json:"name"`
	Url        string            `json:"url"`
	ResHeaders map[string]string `json:"headers"`
}
type webhooksRes struct {
	Webhooks []webhookRes `json:"webhooks"`
}

func TestListWebhooksByGroup(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	whs, err := svc.CreateWebhooks(context.Background(), token, webhook)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	wh := whs[0]

	var data []webhookRes

	for _, webhook := range whs {
		whRes := webhookRes{
			ID:         webhook.ID,
			GroupID:    webhook.GroupID,
			Name:       webhook.Name,
			Url:        webhook.Url,
			ResHeaders: webhook.Headers,
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
			desc:   "view webhooks by group",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/groups/%s/webhooks", ts.URL, wh.GroupID),
			res:    data,
		},
		{
			desc:   "view webhooks by group with invalid token",
			auth:   wrongValue,
			status: http.StatusForbidden,
			url:    fmt.Sprintf("%s/groups/%s/webhooks", ts.URL, wh.GroupID),
			res:    []webhookRes{},
		},
		{
			desc:   "view webhooks by group with empty token",
			auth:   emptyValue,
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s/groups/%s/webhooks", ts.URL, wh.GroupID),
			res:    []webhookRes{},
		},
		{
			desc:   "view webhooks by group with invalid thing id",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/groups/%s/webhooks", ts.URL, wrongValue),
			res:    []webhookRes{},
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
		var data webhooksRes
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
		GroupID:    wh.GroupID,
		Name:       wh.Name,
		Url:        wh.Url,
		ResHeaders: wh.Headers,
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
			url:         fmt.Sprintf("%s/groups/%s/webhooks", ts.URL, groupID),
			token:       tc.auth,
			contentType: tc.contentType,
			body:        strings.NewReader(body),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}
