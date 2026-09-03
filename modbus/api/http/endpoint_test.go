// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/modbus"
	httpapi "github.com/MainfluxLabs/mainflux/modbus/api/http"
	mbmocks "github.com/MainfluxLabs/mainflux/modbus/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/cron"
	pkgmocks "github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	token        = "admin@example.com"
	wrongToken   = "wrong-token"
	emptyValue   = ""
	contentType  = "application/json"
	thingID      = "5384fb1c-d0ae-4cbe-be52-c54223150fe0"
	groupID      = "574106f7-030e-4881-8ab0-151195c29f94"
	wrongID      = "wrong-id"
	testIP       = "192.168.1.1"
	testPort     = "502"
	testFuncCode = "ReadHoldingRegisters"

	missingPart  = "__missing__"
	csvFileName  = "registers.csv"
	jsonFileName = "registers.json"
)

var (
	validScheduler   = `"scheduler":{"frequency":"minutely","minute":5,"time_zone":"UTC"}`
	invalidScheduler = `"scheduler":{"frequency":"invalid"}`

	dataField       = `{"name":"temperature","type":"float32","byte_order":"ABCD","address":0}`
	validDataFields = fmt.Sprintf(`"data_fields":[%s]`, dataField)

	// baseClientJSON holds the fixed name/IP/port/function_code fields shared by most test bodies.
	baseClientJSON  = fmt.Sprintf(`"name":"test-client","ip_address":"%s","port":"%s","function_code":"%s"`, testIP, testPort, testFuncCode)
	validCreateBody = fmt.Sprintf(`[{%s,%s,%s}]`, baseClientJSON, validScheduler, validDataFields)
	validUpdateBody = fmt.Sprintf(`{"name":"updated-client","ip_address":"%s","port":"%s","function_code":"%s",%s,%s}`,
		testIP, testPort, testFuncCode, validScheduler, validDataFields)

	testClient = modbus.Client{
		Name:         "test-client",
		IPAddress:    testIP,
		Port:         testPort,
		SlaveID:      1,
		FunctionCode: modbus.ReadHoldingRegistersFunc,
		Scheduler: cron.Scheduler{
			Frequency: cron.MinutelyFreq,
			Minute:    5,
			TimeZone:  "UTC",
		},
		DataFields: []modbus.DataField{
			{
				Name:      "temperature",
				Type:      modbus.Float32Type,
				ByteOrder: modbus.ByteOrderABCD,
				Address:   0,
			},
		},
	}
)

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

type clientRes struct {
	ID           string         `json:"id"`
	GroupID      string         `json:"group_id"`
	ThingID      string         `json:"thing_id"`
	Name         string         `json:"name"`
	IPAddress    string         `json:"ip_address"`
	Port         string         `json:"port"`
	FunctionCode string         `json:"function_code"`
	Scheduler    cron.Scheduler `json:"scheduler"`
}

type clientsPageRes struct {
	Clients []clientRes `json:"clients"`
	Total   uint64      `json:"total"`
	Offset  uint64      `json:"offset"`
	Limit   uint64      `json:"limit"`
}

func toJSON(data any) string {
	b, _ := json.Marshal(data)
	return string(b)
}

func newService() modbus.Service {
	thingsSvc := pkgmocks.NewThingsServiceClient(
		nil,
		map[string]things.Thing{
			token:   {ID: thingID, GroupID: groupID},
			thingID: {ID: thingID, GroupID: groupID},
		},
		map[string]things.Group{token: {ID: groupID}},
	)
	repo := mbmocks.NewClientRepository()
	pub := pkgmocks.NewPublisher()
	idp := uuid.NewMock()
	log := logger.NewMock()

	return modbus.New(thingsSvc, pub, repo, idp, log)
}

func newHTTPServer(svc modbus.Service) *httptest.Server {
	ac := pkgmocks.NewAuthService("", nil, nil)
	mux := httpapi.MakeHandler(mocktracer.New(), svc, ac, logger.NewMock())
	return httptest.NewServer(mux)
}

func TestCreateClients(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	longName := strings.Repeat("a", 255)

	withField := func(field string) string {
		return fmt.Sprintf(`[{%s,%s,"data_fields":[%s]}]`, baseClientJSON, validScheduler, field)
	}

	cases := []struct {
		desc        string
		body        string
		thingID     string
		contentType string
		token       string
		status      int
	}{
		{
			desc:        "create clients with valid request",
			body:        validCreateBody,
			thingID:     thingID,
			contentType: contentType,
			token:       token,
			status:      http.StatusCreated,
		},
		{
			desc:        "create clients without content type",
			body:        validCreateBody,
			thingID:     thingID,
			contentType: emptyValue,
			token:       token,
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "create clients with invalid JSON",
			body:        `}{`,
			thingID:     thingID,
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "create clients with empty JSON array",
			body:        `[]`,
			thingID:     thingID,
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "create clients with empty name",
			body:        fmt.Sprintf(`[{"name":"","ip_address":"%s","port":"%s","function_code":"%s",%s,%s}]`, testIP, testPort, testFuncCode, validScheduler, validDataFields),
			thingID:     thingID,
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "create clients with name too long",
			body:        fmt.Sprintf(`[{"name":"%s","ip_address":"%s","port":"%s","function_code":"%s",%s,%s}]`, longName, testIP, testPort, testFuncCode, validScheduler, validDataFields),
			thingID:     thingID,
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "create clients with missing IP address",
			body:        fmt.Sprintf(`[{"name":"test-client","ip_address":"","port":"%s","function_code":"%s",%s,%s}]`, testPort, testFuncCode, validScheduler, validDataFields),
			thingID:     thingID,
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "create clients with missing port",
			body:        fmt.Sprintf(`[{"name":"test-client","ip_address":"%s","port":"","function_code":"%s",%s,%s}]`, testIP, testFuncCode, validScheduler, validDataFields),
			thingID:     thingID,
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "create clients with invalid scheduler",
			body:        fmt.Sprintf(`[{%s,%s,%s}]`, baseClientJSON, invalidScheduler, validDataFields),
			thingID:     thingID,
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "create clients with invalid function code",
			body:        fmt.Sprintf(`[{"name":"test-client","ip_address":"%s","port":"%s","function_code":"invalid",%s,%s}]`, testIP, testPort, validScheduler, validDataFields),
			thingID:     thingID,
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "create clients with empty data fields",
			body:        fmt.Sprintf(`[{%s,%s,"data_fields":[]}]`, baseClientJSON, validScheduler),
			thingID:     thingID,
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "create clients with data field missing name",
			body:        withField(`{"name":"","type":"float32","byte_order":"ABCD","address":0}`),
			thingID:     thingID,
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "create clients with data field invalid type",
			body:        withField(`{"name":"temperature","type":"invalid","byte_order":"ABCD","address":0}`),
			thingID:     thingID,
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "create clients with data field invalid byte order",
			body:        withField(`{"name":"temperature","type":"float32","byte_order":"invalid","address":0}`),
			thingID:     thingID,
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "create clients with string field missing length",
			body:        withField(`{"name":"label","type":"string","byte_order":"ABCD","address":0,"length":0}`),
			thingID:     thingID,
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "create clients with wrong token",
			body:        validCreateBody,
			thingID:     thingID,
			contentType: contentType,
			token:       wrongToken,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "create clients with empty token",
			body:        validCreateBody,
			thingID:     thingID,
			contentType: contentType,
			token:       emptyValue,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "create clients with wrong thing ID",
			body:        validCreateBody,
			thingID:     wrongID,
			contentType: contentType,
			token:       token,
			status:      http.StatusForbidden,
		},
		{
			desc:        "create clients with empty thing ID",
			body:        validCreateBody,
			thingID:     emptyValue,
			contentType: contentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/things/%s/clients", ts.URL, tc.thingID),
			contentType: tc.contentType,
			token:       tc.token,
			body:        strings.NewReader(tc.body),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestListClientsByThing(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	cls, err := svc.CreateClients(context.Background(), token, thingID, testClient)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	require.Equal(t, 1, len(cls))

	cases := []struct {
		desc   string
		token  string
		url    string
		status int
		size   int
	}{
		{
			desc:   "list clients by thing",
			token:  token,
			url:    fmt.Sprintf("%s/things/%s/clients", ts.URL, thingID),
			status: http.StatusOK,
			size:   1,
		},
		{
			desc:   "list clients by thing with wrong token",
			token:  wrongToken,
			url:    fmt.Sprintf("%s/things/%s/clients", ts.URL, thingID),
			status: http.StatusUnauthorized,
			size:   0,
		},
		{
			desc:   "list clients by thing with empty token",
			token:  emptyValue,
			url:    fmt.Sprintf("%s/things/%s/clients", ts.URL, thingID),
			status: http.StatusUnauthorized,
			size:   0,
		},
		{
			desc:   "list clients by thing with wrong thing ID",
			token:  token,
			url:    fmt.Sprintf("%s/things/%s/clients", ts.URL, wrongID),
			status: http.StatusForbidden,
			size:   0,
		},
		{
			desc:   "list clients by thing with negative offset",
			token:  token,
			url:    fmt.Sprintf("%s/things/%s/clients?offset=-1&limit=5", ts.URL, thingID),
			status: http.StatusBadRequest,
			size:   0,
		},
		{
			desc:   "list clients by thing with invalid limit",
			token:  token,
			url:    fmt.Sprintf("%s/things/%s/clients?offset=0&limit=abc", ts.URL, thingID),
			status: http.StatusBadRequest,
			size:   0,
		},
		{
			desc:   "list clients by thing with limit exceeding maximum",
			token:  token,
			url:    fmt.Sprintf("%s/things/%s/clients?offset=0&limit=201", ts.URL, thingID),
			status: http.StatusBadRequest,
			size:   0,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    tc.url,
			token:  tc.token,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d", tc.desc, tc.status, res.StatusCode))

		var body clientsPageRes
		json.NewDecoder(res.Body).Decode(&body)
		assert.Equal(t, tc.size, len(body.Clients), fmt.Sprintf("%s: expected %d clients got %d", tc.desc, tc.size, len(body.Clients)))
	}
}

func TestListClientsByGroup(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	_, err := svc.CreateClients(context.Background(), token, thingID, testClient)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc   string
		token  string
		url    string
		status int
		size   int
	}{
		{
			desc:   "list clients by group",
			token:  token,
			url:    fmt.Sprintf("%s/groups/%s/clients", ts.URL, groupID),
			status: http.StatusOK,
			size:   1,
		},
		{
			desc:   "list clients by group with wrong token",
			token:  wrongToken,
			url:    fmt.Sprintf("%s/groups/%s/clients", ts.URL, groupID),
			status: http.StatusUnauthorized,
			size:   0,
		},
		{
			desc:   "list clients by group with empty token",
			token:  emptyValue,
			url:    fmt.Sprintf("%s/groups/%s/clients", ts.URL, groupID),
			status: http.StatusUnauthorized,
			size:   0,
		},
		{
			desc:   "list clients by group with wrong group ID",
			token:  token,
			url:    fmt.Sprintf("%s/groups/%s/clients", ts.URL, wrongID),
			status: http.StatusForbidden,
			size:   0,
		},
		{
			desc:   "list clients by group with negative offset",
			token:  token,
			url:    fmt.Sprintf("%s/groups/%s/clients?offset=-1&limit=5", ts.URL, groupID),
			status: http.StatusBadRequest,
			size:   0,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    tc.url,
			token:  tc.token,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d", tc.desc, tc.status, res.StatusCode))

		var body clientsPageRes
		json.NewDecoder(res.Body).Decode(&body)
		assert.Equal(t, tc.size, len(body.Clients), fmt.Sprintf("%s: expected %d clients got %d", tc.desc, tc.size, len(body.Clients)))
	}
}

func TestViewClient(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	cls, err := svc.CreateClients(context.Background(), token, thingID, testClient)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	clID := cls[0].ID

	cases := []struct {
		desc   string
		token  string
		id     string
		status int
	}{
		{
			desc:   "view client",
			token:  token,
			id:     clID,
			status: http.StatusOK,
		},
		{
			desc:   "view client with non-existent ID",
			token:  token,
			id:     wrongID,
			status: http.StatusNotFound,
		},
		{
			desc:   "view client with wrong token",
			token:  wrongToken,
			id:     clID,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "view client with empty token",
			token:  emptyValue,
			id:     clID,
			status: http.StatusUnauthorized,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/clients/%s", ts.URL, tc.id),
			token:  tc.token,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d", tc.desc, tc.status, res.StatusCode))

		if tc.status == http.StatusOK {
			var body clientRes
			json.NewDecoder(res.Body).Decode(&body)
			assert.Equal(t, clID, body.ID, fmt.Sprintf("%s: expected ID %s got %s", tc.desc, clID, body.ID))
		}
	}
}

func TestUpdateClient(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	cls, err := svc.CreateClients(context.Background(), token, thingID, testClient)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	clID := cls[0].ID

	cases := []struct {
		desc        string
		token       string
		id          string
		body        string
		contentType string
		status      int
	}{
		{
			desc:        "update client with valid request",
			token:       token,
			id:          clID,
			body:        validUpdateBody,
			contentType: contentType,
			status:      http.StatusOK,
		},
		{
			desc:        "update client without content type",
			token:       token,
			id:          clID,
			body:        validUpdateBody,
			contentType: emptyValue,
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "update client with invalid JSON",
			token:       token,
			id:          clID,
			body:        `}{`,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update client with non-existent ID",
			token:       token,
			id:          wrongID,
			body:        validUpdateBody,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "update client with wrong token",
			token:       wrongToken,
			id:          clID,
			body:        validUpdateBody,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update client with empty token",
			token:       emptyValue,
			id:          clID,
			body:        validUpdateBody,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update client with invalid scheduler",
			token:       token,
			id:          clID,
			body:        fmt.Sprintf(`{%s,%s,%s}`, baseClientJSON, invalidScheduler, validDataFields),
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update client with invalid function code",
			token:       token,
			id:          clID,
			body:        fmt.Sprintf(`{"name":"test-client","ip_address":"%s","port":"%s","function_code":"invalid",%s,%s}`, testIP, testPort, validScheduler, validDataFields),
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPut,
			url:         fmt.Sprintf("%s/clients/%s", ts.URL, tc.id),
			token:       tc.token,
			body:        strings.NewReader(tc.body),
			contentType: tc.contentType,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestRemoveClients(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	cls, err := svc.CreateClients(context.Background(), token, thingID, testClient)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	clID := cls[0].ID

	removeBody := toJSON(map[string]any{"client_ids": []string{clID}})

	cases := []struct {
		desc        string
		token       string
		body        string
		contentType string
		status      int
	}{
		{
			desc:        "remove clients with non-existent ID",
			token:       token,
			body:        toJSON(map[string]any{"client_ids": []string{wrongID}}),
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "remove clients without content type",
			token:       token,
			body:        removeBody,
			contentType: emptyValue,
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "remove clients with invalid JSON",
			token:       token,
			body:        `}{`,
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "remove clients with empty ID list",
			token:       token,
			body:        toJSON(map[string]any{"client_ids": []string{}}),
			contentType: contentType,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "remove clients with wrong token",
			token:       wrongToken,
			body:        removeBody,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "remove clients with empty token",
			token:       emptyValue,
			body:        removeBody,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "remove clients with valid request",
			token:       token,
			body:        removeBody,
			contentType: contentType,
			status:      http.StatusNoContent,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/clients", ts.URL),
			token:       tc.token,
			body:        strings.NewReader(tc.body),
			contentType: tc.contentType,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func multipartBody(t *testing.T, clientJSON, fileName, fileContent string) (*bytes.Buffer, string) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	if clientJSON != missingPart {
		require.NoError(t, w.WriteField("client", clientJSON))
	}

	if fileName != missingPart {
		fw, err := w.CreateFormFile("file", fileName)
		require.NoError(t, err)
		_, err = fw.Write([]byte(fileContent))
		require.NoError(t, err)
	}

	require.NoError(t, w.Close())

	return &buf, w.FormDataContentType()
}

func TestCreateClientFromCSV(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	validClientJSON := fmt.Sprintf(`{%s,%s}`, baseClientJSON, validScheduler)
	validFieldsCSV := "name,type,unit,scale,byte_order,address,length\ntemperature,float32,,,ABCD,0,\nhumidity,float32,,,ABCD,2,\n"

	validBody, validContentType := multipartBody(t, validClientJSON, csvFileName, validFieldsCSV)
	badRowBody, badRowContentType := multipartBody(t, validClientJSON, csvFileName,
		"name,type,unit,scale,byte_order,address,length\ntemperature,float32,,,ABCD,not-a-number,\n")
	emptyFileBody, emptyFileContentType := multipartBody(t, validClientJSON, csvFileName, "")
	badClientJSONBody, badClientJSONContentType := multipartBody(t, `}{`, csvFileName, validFieldsCSV)
	noClientPartBody, noClientPartContentType := multipartBody(t, missingPart, csvFileName, validFieldsCSV)
	noFilePartBody, noFilePartContentType := multipartBody(t, validClientJSON, missingPart, "")
	wrongTokenBody, wrongTokenContentType := multipartBody(t, validClientJSON, csvFileName, validFieldsCSV)
	wrongThingBody, wrongThingContentType := multipartBody(t, validClientJSON, csvFileName, validFieldsCSV)

	cases := []struct {
		desc        string
		body        *bytes.Buffer
		thingID     string
		contentType string
		token       string
		status      int
	}{
		{
			desc:        "import client with valid csv file",
			body:        validBody,
			thingID:     thingID,
			contentType: validContentType,
			token:       token,
			status:      http.StatusCreated,
		},
		{
			desc:        "import client with unsupported content type",
			body:        validBody,
			thingID:     thingID,
			contentType: "text/plain",
			token:       token,
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "import client with malformed csv row",
			body:        badRowBody,
			thingID:     thingID,
			contentType: badRowContentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "import client with empty file",
			body:        emptyFileBody,
			thingID:     thingID,
			contentType: emptyFileContentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "import client with invalid client JSON",
			body:        badClientJSONBody,
			thingID:     thingID,
			contentType: badClientJSONContentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "import client with missing client part",
			body:        noClientPartBody,
			thingID:     thingID,
			contentType: noClientPartContentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "import client with missing file part",
			body:        noFilePartBody,
			thingID:     thingID,
			contentType: noFilePartContentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "import client with wrong token",
			body:        wrongTokenBody,
			thingID:     thingID,
			contentType: wrongTokenContentType,
			token:       wrongToken,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "import client with wrong thing ID",
			body:        wrongThingBody,
			thingID:     wrongID,
			contentType: wrongThingContentType,
			token:       token,
			status:      http.StatusForbidden,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/things/%s/clients/csv", ts.URL, tc.thingID),
			contentType: tc.contentType,
			token:       tc.token,
			body:        tc.body,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d", tc.desc, tc.status, res.StatusCode))

		if tc.status == http.StatusCreated {
			var body clientRes
			err = json.NewDecoder(res.Body).Decode(&body)
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.NotEmpty(t, body.ID, fmt.Sprintf("%s: expected a single client object with an id", tc.desc))
		}
	}
}

func TestCreateClientFromJSON(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	validClientJSON := fmt.Sprintf(`{%s,%s}`, baseClientJSON, validScheduler)
	validFieldsJSON := fmt.Sprintf(`[%s]`, dataField)

	validBody, validContentType := multipartBody(t, validClientJSON, jsonFileName, validFieldsJSON)
	badFieldsBody, badFieldsContentType := multipartBody(t, validClientJSON, jsonFileName, `not valid json`)
	emptyFileBody, emptyFileContentType := multipartBody(t, validClientJSON, jsonFileName, "")
	badClientJSONBody, badClientJSONContentType := multipartBody(t, `}{`, jsonFileName, validFieldsJSON)
	noClientPartBody, noClientPartContentType := multipartBody(t, missingPart, jsonFileName, validFieldsJSON)
	noFilePartBody, noFilePartContentType := multipartBody(t, validClientJSON, missingPart, "")
	wrongTokenBody, wrongTokenContentType := multipartBody(t, validClientJSON, jsonFileName, validFieldsJSON)
	wrongThingBody, wrongThingContentType := multipartBody(t, validClientJSON, jsonFileName, validFieldsJSON)

	cases := []struct {
		desc        string
		body        *bytes.Buffer
		thingID     string
		contentType string
		token       string
		status      int
	}{
		{
			desc:        "import client with valid json file",
			body:        validBody,
			thingID:     thingID,
			contentType: validContentType,
			token:       token,
			status:      http.StatusCreated,
		},
		{
			desc:        "import client with unsupported content type",
			body:        validBody,
			thingID:     thingID,
			contentType: "text/plain",
			token:       token,
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "import client with malformed json fields",
			body:        badFieldsBody,
			thingID:     thingID,
			contentType: badFieldsContentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "import client with empty file",
			body:        emptyFileBody,
			thingID:     thingID,
			contentType: emptyFileContentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "import client with invalid client JSON",
			body:        badClientJSONBody,
			thingID:     thingID,
			contentType: badClientJSONContentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "import client with missing client part",
			body:        noClientPartBody,
			thingID:     thingID,
			contentType: noClientPartContentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "import client with missing file part",
			body:        noFilePartBody,
			thingID:     thingID,
			contentType: noFilePartContentType,
			token:       token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "import client with wrong token",
			body:        wrongTokenBody,
			thingID:     thingID,
			contentType: wrongTokenContentType,
			token:       wrongToken,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "import client with wrong thing ID",
			body:        wrongThingBody,
			thingID:     wrongID,
			contentType: wrongThingContentType,
			token:       token,
			status:      http.StatusForbidden,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPost,
			url:         fmt.Sprintf("%s/things/%s/clients/json", ts.URL, tc.thingID),
			contentType: tc.contentType,
			token:       tc.token,
			body:        tc.body,
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status %d got %d", tc.desc, tc.status, res.StatusCode))

		if tc.status == http.StatusCreated {
			var body clientRes
			err = json.NewDecoder(res.Body).Decode(&body)
			assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
			assert.NotEmpty(t, body.ID, fmt.Sprintf("%s: expected a single client object with an id", tc.desc))
		}
	}
}
