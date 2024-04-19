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

	"github.com/MainfluxLabs/mainflux/internal/apiutil"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
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
	thingID     = "50e6b371-60ff-45cf-bb52-8200e7cde536"
)

func newHTTPServer(svc webhooks.Service) *httptest.Server {
	logger := logger.NewMock()
	mux := httpapi.MakeHandler(mocktracer.New(), svc, logger)
	return httptest.NewServer(mux)
}

func newService() webhooks.Service {
	things := mocks.NewThingsServiceClient(nil, map[string]string{token: thingID}, nil)
	webhookRepo := whmocks.NewWebhookRepository()
	forwarder := whmocks.NewForwarder()

	return webhooks.New(things, webhookRepo, forwarder)
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

	validData := `[{"name":"value","value_fields":"["value1","value2"]","url":"https://api.example.com"}]`
	invalidName := fmt.Sprintf(`[{"name":"%s","value_fields":"["value1","value2"]","url":"https://api.example.com"}]`, emptyValue)
	invalidUrl := fmt.Sprintf(`[{"name":"value","value_fields":"["value1","value2"]","url":"%s"}]`, invalidUrl)

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
			status:      http.StatusForbidden,
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
			url:         fmt.Sprintf("%s/webhooks/%s", ts.URL, tc.thingID),
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
	ThingID     string   `json:"thing_id"`
	Name        string   `json:"name"`
	ValueFields []string `json:"value_fields"`
	Url         string   `json:"url"`
}
type webhooksRes struct {
	Webhooks []webhookRes `json:"webhooks"`
}

func TestListWebhooksByThing(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	formatter := webhooks.Formatter{Fields: []string{"value1", "value2"}}
	webhook := webhooks.Webhook{
		ThingID:   "50e6b371-60ff-45cf-bb52-8200e7cde536",
		Name:      "test-webhook",
		Formatter: formatter,
		Url:       "https://test.webhook.com",
	}

	whs, err := svc.CreateWebhooks(context.Background(), token, webhook)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	wh := whs[0]

	var data []webhookRes

	for _, webhook := range whs {
		whRes := webhookRes{
			ThingID:     webhook.ThingID,
			Name:        webhook.Name,
			ValueFields: webhook.Formatter.Fields,
			Url:         webhook.Url,
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
			desc:   "view webhooks by thing",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/webhooks/%s", ts.URL, wh.ThingID),
			res:    data,
		},
		{
			desc:   "view webhooks by thing with invalid token",
			auth:   wrongValue,
			status: http.StatusForbidden,
			url:    fmt.Sprintf("%s/webhooks/%s", ts.URL, wh.ThingID),
			res:    []webhookRes{},
		},
		{
			desc:   "view webhooks by thing with empty token",
			auth:   emptyValue,
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s/webhooks/%s", ts.URL, wh.ThingID),
			res:    []webhookRes{},
		},
		{
			desc:   "view webhooks by thing with invalid thing id",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/webhooks/%s", ts.URL, wrongValue),
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
