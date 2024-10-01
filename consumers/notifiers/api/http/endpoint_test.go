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

	"github.com/MainfluxLabs/mainflux/consumers/notifiers"
	httpapi "github.com/MainfluxLabs/mainflux/consumers/notifiers/api/http"
	ntmocks "github.com/MainfluxLabs/mainflux/consumers/notifiers/mocks"
	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/opentracing/opentracing-go/mocktracer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	token        = "admin@example.com"
	wrongValue   = "wrong-value"
	contentType  = "application/json"
	emptyValue   = ""
	groupID      = "50e6b371-60ff-45cf-bb52-8200e7cde536"
	prefixID     = "fe6b4e92-cc98-425e-b0aa-"
	prefixName   = "test-notifier-"
	notifierName = "notifier-test"
	svcSmtp      = "smtp-notifier"
	svcSmpp      = "smpp-notifier"
	nameKey      = "name"
	ascKey       = "asc"
	descKey      = "desc"
)

var (
	validEmails     = []string{"user1@example.com", "user2@example.com"}
	validPhones     = []string{"+381610120120", "+381622220123"}
	invalidEmails   = []string{"invalid@example.com", "invalid@invalid"}
	invalidPhones   = []string{"0610120120", "0622220123"}
	metadata        = map[string]interface{}{"test": "data"}
	missingIDRes    = toJSON(apiutil.ErrorRes{Err: apiutil.ErrMissingID.Error()})
	missingTokenRes = toJSON(apiutil.ErrorRes{Err: apiutil.ErrBearerToken.Error()})
)

func newHTTPServer(svc notifiers.Service) *httptest.Server {
	logger := logger.NewMock()
	mux := httpapi.MakeHandler(mocktracer.New(), svc, logger)
	return httptest.NewServer(mux)
}

func toJSON(data interface{}) string {
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

func newService() notifiers.Service {
	things := mocks.NewThingsServiceClient(nil, nil, map[string]things.Group{token: {ID: groupID}})
	notifier := ntmocks.NewNotifier()
	notifierRepo := ntmocks.NewNotifierRepository()
	idp := uuid.NewMock()
	return notifiers.New(idp, notifier, notifierRepo, things)
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

func TestCreateNotifiers(t *testing.T) {
	runCreateNotifiersTest(t, validEmails[0])
	runCreateNotifiersTest(t, validPhones[0])
}

func runCreateNotifiersTest(t *testing.T, validContacts string) {
	t.Helper()
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	var invalidContactsData string
	validData := fmt.Sprintf(`[{"name":"%s","contacts":["%s"]}]`, notifierName, validContacts)
	invalidNameData := fmt.Sprintf(`[{"name":"","contacts":["%s"]}]`, validContacts)
	invalidContactsData = fmt.Sprintf(`[{"name":"%s","contacts":[]}]`, notifierName)

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
			desc:        "create valid notifiers",
			data:        validData,
			groupID:     groupID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusCreated,
			response:    emptyValue,
		},
		{
			desc:        "create notifiers with empty request",
			data:        emptyValue,
			groupID:     groupID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    emptyValue,
		},
		{
			desc:        "create notifiers with invalid request format",
			data:        "}{",
			groupID:     groupID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    emptyValue,
		},
		{
			desc:        "create notifiers with invalid group id",
			data:        validData,
			groupID:     emptyValue,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    emptyValue,
		},
		{
			desc:        "create notifiers with invalid contacts",
			data:        invalidContactsData,
			groupID:     groupID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    emptyValue,
		},
		{
			desc:        "create notifiers with invalid name",
			data:        invalidNameData,
			groupID:     groupID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    emptyValue,
		},
		{
			desc:        "create notifiers with empty JSON array",
			data:        "[]",
			groupID:     groupID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
			response:    emptyValue,
		},
		{
			desc:        "create notifier with wrong auth token",
			data:        validData,
			groupID:     groupID,
			contentType: contentType,
			auth:        wrongValue,
			status:      http.StatusUnauthorized,
			response:    emptyValue,
		},
		{
			desc:        "create notifier with empty auth token",
			data:        validData,
			groupID:     groupID,
			contentType: contentType,
			auth:        emptyValue,
			status:      http.StatusUnauthorized,
			response:    emptyValue,
		},
		{
			desc:        "create notifier without content type",
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
			url:         fmt.Sprintf("%s/groups/%s/notifiers", ts.URL, tc.groupID),
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

type notifierRes struct {
	ID       string                 `json:"id"`
	GroupID  string                 `json:"group_id"`
	Name     string                 `json:"name"`
	Contacts []string               `json:"contacts"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type notifiersPageRes struct {
	Notifiers []notifierRes `json:"notifiers"`
	Total     uint64        `json:"total"`
	Offset    uint64        `json:"offset"`
	Limit     uint64        `json:"limit"`
}

func TestListNotifiersByGroup(t *testing.T) {
	runListNotifiersByGroupTest(t, validEmails)
	runListNotifiersByGroupTest(t, validPhones)
}

func runListNotifiersByGroupTest(t *testing.T, validContacts []string) {
	t.Helper()
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	var data []notifierRes
	notifier := things.Notifier{GroupID: groupID, Name: "test-notifier", Contacts: validContacts, Metadata: metadata}

	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("%s%012d", prefixID, i+1)
		name := fmt.Sprintf("%s%012d", prefixName, i+1)
		notifier1 := notifier
		notifier1.ID = id
		notifier1.Name = name

		nfs, err := svc.CreateNotifiers(context.Background(), token, notifier1)
		require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
		n := nfs[0]
		nfRes := notifierRes{
			ID:       n.ID,
			GroupID:  n.GroupID,
			Name:     n.Name,
			Contacts: n.Contacts,
			Metadata: n.Metadata,
		}
		data = append(data, nfRes)
	}

	cases := []struct {
		desc   string
		auth   string
		status int
		url    string
		res    []notifierRes
	}{
		{
			desc:   "get a list of notifiers by group",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/groups/%s/notifiers?offset=%d&limit=%d", ts.URL, groupID, 0, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of notifiers by group with no limit",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/groups/%s/notifiers?limit=%d", ts.URL, groupID, -1),
			res:    data,
		},
		{
			desc:   "get a list of notifiers by group with negative offset",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/groups/%s/notifiers?offset=%d&limit=%d", ts.URL, groupID, -2, 2),
			res:    nil,
		},
		{
			desc:   "get a list of notifiers by group with negative limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/groups/%s/notifiers?offset=%d&limit=%d", ts.URL, groupID, 2, -3),
			res:    nil,
		},
		{
			desc:   "get a list of notifiers by group with zero limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/groups/%s/notifiers?offset=%d&limit=%d", ts.URL, groupID, 2, 0),
			res:    nil,
		},
		{
			desc:   "get a list of notifiers by group with invalid token",
			auth:   wrongValue,
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s/groups/%s/notifiers?offset=%d&limit=%d", ts.URL, groupID, 0, 1),
			res:    nil,
		},
		{
			desc:   "get a list of notifiers by group with empty token",
			auth:   emptyValue,
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s/groups/%s/notifiers?offset=%d&limit=%d", ts.URL, groupID, 0, 1),
			res:    []notifierRes{},
		},
		{
			desc:   "get a list of notifiers by group with wrong group id",
			auth:   token,
			status: http.StatusForbidden,
			url:    fmt.Sprintf("%s/groups/%s/notifiers", ts.URL, wrongValue),
			res:    []notifierRes{},
		},
		{
			desc:   "get a list of notifiers by group without offset",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/groups/%s/notifiers?limit=%d", ts.URL, groupID, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of notifiers by group without limit",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/groups/%s/notifiers?offset=%d", ts.URL, groupID, 1),
			res:    data[1:10],
		},
		{
			desc:   "get a list of notifiers by group with redundant query params",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/groups/%s/notifiers?offset=%d&limit=%d&value=something", ts.URL, groupID, 0, 5),
			res:    data[0:5],
		},
		{
			desc:   "get a list of notifiers by group with limit greater than max",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/groups/%s/notifiers?offset=%d&limit=%d", ts.URL, groupID, 0, 101),
			res:    nil,
		},
		{
			desc:   "get a list of notifiers by group with default URL",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/groups/%s/notifiers", ts.URL, groupID),
			res:    data[0:10],
		},
		{
			desc:   "get a list of notifiers by group with invalid number of params",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/groups/%s/notifiers%s", ts.URL, groupID, "?offset=4&limit=4&limit=5&offset=5"),
			res:    nil,
		},
		{
			desc:   "get a list of notifiers by group with invalid offset",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/groups/%s/notifiers%s", ts.URL, groupID, "?offset=e&limit=5"),
			res:    nil,
		},
		{
			desc:   "get a list of notifiers by group with invalid limit",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/groups/%s/notifiers%s", ts.URL, groupID, "?offset=5&limit=e"),
			res:    nil,
		},
		{
			desc:   "get a list of notifiers by group sorted by name ascendant",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/groups/%s/notifiers?offset=%d&limit=%d&order=%s&dir=%s", ts.URL, groupID, 0, 5, nameKey, ascKey),
			res:    data[0:5],
		},
		{
			desc:   "get a list of notifiers by group sorted by name descendent",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/groups/%s/notifiers?offset=%d&limit=%d&order=%s&dir=%s", ts.URL, groupID, 0, 5, nameKey, descKey),
			res:    data[0:5],
		},
		{
			desc:   "get a list of notifiers by group sorted with invalid order",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/groups/%s/notifiers?offset=%d&limit=%d&order=%s&dir=%s", ts.URL, groupID, 0, 5, wrongValue, ascKey),
			res:    nil,
		},
		{
			desc:   "get a list of notifiers by group sorted by name with invalid direction",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/groups/%s/notifiers?offset=%d&limit=%d&order=%s&dir=%s", ts.URL, groupID, 0, 5, nameKey, wrongValue),
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
		var data notifiersPageRes
		json.NewDecoder(res.Body).Decode(&data)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res, data.Notifiers, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, data.Notifiers))

	}
}

func TestUpdateNotifier(t *testing.T) {
	runUpdateNotifierTest(t, svcSmtp, validEmails)
	runUpdateNotifierTest(t, svcSmpp, validPhones)
}

func runUpdateNotifierTest(t *testing.T, svcName string, validContacts []string) {
	t.Helper()
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	notifier := things.Notifier{GroupID: groupID, Name: notifierName, Contacts: validContacts, Metadata: metadata}
	data := toJSON(notifier)

	nfs, err := svc.CreateNotifiers(context.Background(), token, notifier)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	nf := nfs[0]

	invalidContactsNf := notifier
	if svcName == svcSmtp {
		invalidContactsNf.Contacts = invalidEmails
	}
	if svcName == svcSmpp {
		invalidContactsNf.Contacts = invalidPhones

	}
	invalidC := toJSON(invalidContactsNf)

	invalidNameNf := notifier
	invalidNameNf.Name = emptyValue
	invalidN := toJSON(invalidNameNf)

	cases := []struct {
		desc        string
		req         string
		id          string
		contentType string
		auth        string
		status      int
	}{
		{
			desc:        "update existing notifier",
			req:         data,
			id:          nf.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusOK,
		},
		{
			desc:        "update non-existent notifier",
			req:         data,
			id:          strconv.FormatUint(0, 10),
			contentType: contentType,
			auth:        token,
			status:      http.StatusNotFound,
		},
		{
			desc:        "update notifier with invalid id",
			req:         data,
			id:          "invalid",
			contentType: contentType,
			auth:        token,
			status:      http.StatusNotFound,
		},
		{
			desc:        "update notifier with empty user token",
			req:         data,
			id:          nf.ID,
			contentType: contentType,
			auth:        emptyValue,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "update notifier with invalid data format",
			req:         "{",
			id:          nf.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update notifier with empty JSON request",
			req:         "{}",
			id:          nf.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update notifier with empty request",
			req:         "",
			id:          nf.ID,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update notifier without content type",
			req:         data,
			id:          nf.ID,
			contentType: "",
			auth:        token,
			status:      http.StatusUnsupportedMediaType,
		},
		{
			desc:        "update notifier with invalid contacts",
			req:         invalidC,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
		{
			desc:        "update notifier with invalid name",
			req:         invalidN,
			contentType: contentType,
			auth:        token,
			status:      http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPut,
			url:         fmt.Sprintf("%s/notifiers/%s", ts.URL, tc.id),
			contentType: tc.contentType,
			token:       tc.auth,
			body:        strings.NewReader(tc.req),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}

func TestViewNotifier(t *testing.T) {
	runViewNotifierTest(t, validEmails)
	runViewNotifierTest(t, validPhones)
}

func runViewNotifierTest(t *testing.T, validContacts []string) {
	t.Helper()
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()
	notifier := things.Notifier{GroupID: groupID, Name: notifierName, Contacts: validContacts, Metadata: metadata}

	nfs, err := svc.CreateNotifiers(context.Background(), token, notifier)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	nf := nfs[0]

	data := toJSON(notifierRes{
		ID:       nf.ID,
		GroupID:  nf.GroupID,
		Name:     nf.Name,
		Contacts: nf.Contacts,
		Metadata: nf.Metadata,
	})

	cases := []struct {
		desc   string
		id     string
		auth   string
		status int
		res    string
	}{
		{
			desc:   "view existing notifier",
			id:     nf.ID,
			auth:   token,
			status: http.StatusOK,
			res:    data,
		},
		{
			desc:   "view notifier with empty token",
			id:     nf.ID,
			auth:   emptyValue,
			status: http.StatusUnauthorized,
			res:    missingTokenRes,
		},
		{
			desc:   "view notifier with invalid id",
			id:     emptyValue,
			auth:   token,
			status: http.StatusBadRequest,
			res:    missingIDRes,
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    fmt.Sprintf("%s/notifiers/%s", ts.URL, tc.id),
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

func TestRemoveNotifiers(t *testing.T) {
	runRemoveNotifiersTest(t, validEmails)
	runRemoveNotifiersTest(t, validPhones)
}

func runRemoveNotifiersTest(t *testing.T, validContacts []string) {
	t.Helper()
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()
	notifier := things.Notifier{GroupID: groupID, Name: notifierName, Contacts: validContacts, Metadata: metadata}

	grNfs, err := svc.CreateNotifiers(context.Background(), token, notifier)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))

	var notifierIDs []string
	for _, nf := range grNfs {
		notifierIDs = append(notifierIDs, nf.ID)
	}

	cases := []struct {
		desc        string
		data        []string
		auth        string
		contentType string
		status      int
	}{
		{
			desc:        "remove existing notifiers",
			data:        notifierIDs,
			auth:        token,
			contentType: contentType,
			status:      http.StatusNoContent,
		},
		{
			desc:        "remove non-existent notifiers",
			data:        []string{wrongValue},
			auth:        token,
			contentType: contentType,
			status:      http.StatusNotFound,
		},
		{
			desc:        "remove notifiers with empty token",
			data:        notifierIDs,
			auth:        emptyValue,
			contentType: contentType,
			status:      http.StatusUnauthorized,
		},
		{
			desc:        "remove notifiers with invalid content type",
			data:        notifierIDs,
			auth:        token,
			contentType: wrongValue,
			status:      http.StatusUnsupportedMediaType,
		},
	}

	for _, tc := range cases {
		data := struct {
			NotifierIDs []string `json:"notifier_ids"`
		}{
			tc.data,
		}

		body := toJSON(data)

		req := testRequest{
			client:      ts.Client(),
			method:      http.MethodPatch,
			url:         fmt.Sprintf("%s/notifiers", ts.URL),
			token:       tc.auth,
			contentType: tc.contentType,
			body:        strings.NewReader(body),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}
