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
	userEmail       = "user@example.com"
	phoneNum        = "+381610120120"
	invalidPhoneNum = "0610120120"
	invalidUser     = "invalid@example.com"
	token           = "admin@example.com"
	wrongValue      = "wrong-value"
	contentType     = "application/json"
	emptyValue      = ""
	groupID         = "50e6b371-60ff-45cf-bb52-8200e7cde536"
)

var (
	validContacts           = []string{userEmail, phoneNum}
	invalidContacts         = []string{invalidUser, invalidPhoneNum}
	validNotifier           = things.Notifier{GroupID: groupID, Contacts: validContacts}
	invalidContactsNotifier = things.Notifier{GroupID: groupID, Contacts: invalidContacts}
	missingIDRes            = toJSON(apiutil.ErrorRes{Err: apiutil.ErrMissingID.Error()})
	missingTokenRes         = toJSON(apiutil.ErrorRes{Err: apiutil.ErrBearerToken.Error()})
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
	things := mocks.NewThingsServiceClient(nil, map[string]string{token: groupID}, nil)
	notifier := ntmocks.NewNotifier()
	notifierRepo := ntmocks.NewNotifierRepository()
	idp := uuid.NewMock()
	from := "exampleFrom"
	return notifiers.New(idp, notifier, from, notifierRepo, things)
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
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	validData := `[{"contacts":["test@gmail.com"]}]`
	invalidData := `[{"contacts":["test.com","0610120120"]}]`

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
			data:        invalidData,
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
			status:      http.StatusForbidden,
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
	ID       string   `json:"id"`
	GroupID  string   `json:"group_id"`
	Contacts []string `json:"contacts"`
}
type notifiersRes struct {
	Notifiers []notifierRes `json:"notifiers"`
}

func TestListNotifiersByGroup(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	nfs, err := svc.CreateNotifiers(context.Background(), token, validNotifier)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	nf := nfs[0]

	var data []notifierRes

	for _, notifier := range nfs {
		nfRes := notifierRes{
			ID:       notifier.ID,
			GroupID:  notifier.GroupID,
			Contacts: notifier.Contacts,
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
			desc:   "list notifiers by group",
			auth:   token,
			status: http.StatusOK,
			url:    fmt.Sprintf("%s/groups/%s/notifiers", ts.URL, nf.GroupID),
			res:    data,
		},
		{
			desc:   "list notifiers by group with invalid token",
			auth:   wrongValue,
			status: http.StatusForbidden,
			url:    fmt.Sprintf("%s/groups/%s/notifiers", ts.URL, nf.GroupID),
			res:    []notifierRes{},
		},
		{
			desc:   "list notifiers by group with empty token",
			auth:   emptyValue,
			status: http.StatusUnauthorized,
			url:    fmt.Sprintf("%s/groups/%s/notifiers", ts.URL, nf.GroupID),
			res:    []notifierRes{},
		},
		{
			desc:   "list notifiers by group with invalid group id",
			auth:   token,
			status: http.StatusBadRequest,
			url:    fmt.Sprintf("%s/groups/%s/notifiers", ts.URL, emptyValue),
			res:    []notifierRes{},
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
		var data notifiersRes
		json.NewDecoder(res.Body).Decode(&data)
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.ElementsMatch(t, tc.res, data.Notifiers, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res, data.Notifiers))

	}
}

func TestUpdateNotifier(t *testing.T) {
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	data := toJSON(validNotifier)

	nfs, err := svc.CreateNotifiers(context.Background(), token, validNotifier)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s\n", err))
	nf := nfs[0]

	invalidC := toJSON(invalidContactsNotifier)

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
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	nfs, err := svc.CreateNotifiers(context.Background(), token, validNotifier)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	nf := nfs[0]

	data := toJSON(notifierRes{
		ID:       nf.ID,
		GroupID:  nf.GroupID,
		Contacts: nf.Contacts,
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
	svc := newService()
	ts := newHTTPServer(svc)
	defer ts.Close()

	grNfs, err := svc.CreateNotifiers(context.Background(), token, validNotifier)
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
			url:         fmt.Sprintf("%s/groups/%s/notifiers", ts.URL, groupID),
			token:       tc.auth,
			contentType: tc.contentType,
			body:        strings.NewReader(body),
		}
		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected status code %d got %d", tc.desc, tc.status, res.StatusCode))
	}
}
