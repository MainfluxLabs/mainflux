// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MainfluxLabs/mainflux/logger"
	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	thmocks "github.com/MainfluxLabs/mainflux/pkg/mocks"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/senml"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/readers"
	"github.com/MainfluxLabs/mainflux/readers/api"
	rmocks "github.com/MainfluxLabs/mainflux/readers/mocks"
	"github.com/MainfluxLabs/mainflux/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	svcName       = "test-service"
	thingToken    = "1"
	userEmail     = "test@example.com"
	adminEmail    = "admin@example.com"
	invalid       = "invalid"
	numOfMessages = 101
	valueFields   = 5
	subtopic      = "topic"
	mqttProt      = "mqtt"
	httpProt      = "http"
	msgName       = "temperature"
	validPass     = "password"
	adminID       = "1"
)

var (
	v   float64 = 5
	vs          = "value"
	vb          = true
	vd          = "dataValue"
	sum float64 = 42

	idProvider = uuid.New()

	user      = users.User{Email: userEmail, Password: validPass}
	admin     = users.User{ID: adminID, Email: adminEmail, Password: validPass, Status: "enabled"}
	usersList = []users.User{user, admin}
)

func newServer(repo readers.MessageRepository, tc protomfx.ThingsServiceClient, ac protomfx.AuthServiceClient) *httptest.Server {
	logger := logger.NewMock()
	mux := api.MakeHandler(repo, tc, ac, svcName, logger)

	id, _ := idProvider.ID()
	user.ID = id

	return httptest.NewServer(mux)
}

type testRequest struct {
	client *http.Client
	method string
	url    string
	token  string
	key    string
}

func (tr testRequest) make() (*http.Response, error) {
	req, err := http.NewRequest(tr.method, tr.url, nil)
	if err != nil {
		return nil, err
	}
	if tr.token != "" {
		req.Header.Set("Authorization", apiutil.BearerPrefix+tr.token)
	}
	if tr.key != "" {
		req.Header.Set("Authorization", apiutil.ThingPrefix+tr.key)
	}

	return tr.client.Do(req)
}
func newAuthService() protomfx.AuthServiceClient {
	return mocks.NewAuthService(admin.ID, usersList)
}

func TestListAllMessages(t *testing.T) {
	pubID, err := idProvider.ID()
	require.Nil(t, err, fmt.Sprintf("got unexpected error: %s", err))

	now := time.Now().Unix()

	var messages []senml.Message
	var queryMsgs []senml.Message
	var valueMsgs []senml.Message
	var boolMsgs []senml.Message
	var stringMsgs []senml.Message
	var dataMsgs []senml.Message

	for i := 0; i < numOfMessages; i++ {
		// Mix possible values as well as value sum.
		msg := senml.Message{
			Publisher: pubID,
			Protocol:  mqttProt,
			Time:      float64(now - int64(i)),
			Name:      "name",
		}

		count := i % valueFields
		switch count {
		case 0:
			msg.Value = &v
			valueMsgs = append(valueMsgs, msg)
		case 1:
			msg.BoolValue = &vb
			boolMsgs = append(boolMsgs, msg)
		case 2:
			msg.StringValue = &vs
			stringMsgs = append(stringMsgs, msg)
		case 3:
			msg.DataValue = &vd
			dataMsgs = append(dataMsgs, msg)
		case 4:
			msg.Sum = &sum
			msg.Subtopic = subtopic
			msg.Protocol = httpProt
			msg.Name = msgName
			queryMsgs = append(queryMsgs, msg)
		}

		messages = append(messages, msg)
	}

	thSvc := thmocks.NewThingsServiceClient(map[string]string{userEmail: ""}, nil, nil)
	authSvc := newAuthService()

	tok, err := authSvc.Issue(context.Background(), &protomfx.IssueReq{Id: user.ID, Email: user.Email, Type: 0})
	require.Nil(t, err, fmt.Sprintf("issue token for user got unexpected error: %s", err))
	adminTok, err := authSvc.Issue(context.Background(), &protomfx.IssueReq{Id: admin.ID, Email: admin.Email})
	require.Nil(t, err, fmt.Sprintf("issue token for admin got unexpected error: %s", err))

	userToken := tok.GetValue()
	adminToken := adminTok.GetValue()

	repo := rmocks.NewMessageRepository("", fromSenml(messages))
	ts := newServer(repo, thSvc, authSvc)
	defer ts.Close()

	cases := []struct {
		desc   string
		req    string
		url    string
		token  string
		key    string
		status int
		res    pageRes
	}{
		{
			desc:   "read all messages page as admin",
			url:    fmt.Sprintf("%s/messages?limit=-1", ts.URL),
			token:  adminToken,
			status: http.StatusOK,
			res: pageRes{
				Total:    uint64(len(messages)),
				Messages: messages,
			},
		},
		{
			desc:   "read all messages page as user",
			url:    fmt.Sprintf("%s/messages?limit=-1", ts.URL),
			token:  userToken,
			status: http.StatusForbidden,
			res:    pageRes{},
		},
		{
			desc:   "read page with valid offset and limit",
			url:    fmt.Sprintf("%s/messages?offset=0&limit=10", ts.URL),
			token:  adminToken,
			status: http.StatusOK,
			res: pageRes{
				Total:    uint64(len(messages)),
				Messages: messages[0:10],
			},
		},
		{
			desc:   "read page with valid offset and limit",
			url:    fmt.Sprintf("%s/messages?offset=0&limit=10", ts.URL),
			token:  adminToken,
			status: http.StatusOK,
			res: pageRes{
				Total:    uint64(len(messages)),
				Messages: messages[0:10],
			},
		},
		{
			desc:   "read page with negative offset",
			url:    fmt.Sprintf("%s/messages?offset=-1&limit=10", ts.URL),
			token:  adminToken,
			status: http.StatusBadRequest,
		},
		{
			desc:   "read page with negative limit",
			url:    fmt.Sprintf("%s/messages?offset=0&limit=-10", ts.URL),
			token:  adminToken,
			status: http.StatusBadRequest,
		},
		{
			desc:   "read page with zero limit",
			url:    fmt.Sprintf("%s/messages?offset=0&limit=0", ts.URL),
			token:  adminToken,
			status: http.StatusBadRequest,
		},
		{
			desc:   "read page with non-integer offset",
			url:    fmt.Sprintf("%s/messages?offset=abc&limit=10", ts.URL),
			token:  adminToken,
			status: http.StatusBadRequest,
		},
		{
			desc:   "read page with non-integer limit",
			url:    fmt.Sprintf("%s/messages?offset=0&limit=abc", ts.URL),
			token:  adminToken,
			status: http.StatusBadRequest,
		},
		{
			desc:   "read page with invalid token",
			url:    fmt.Sprintf("%s/messages?offset=0&limit=10", ts.URL),
			token:  invalid,
			status: http.StatusUnauthorized,
		},
		{
			desc:   "read page with multiple offset",
			url:    fmt.Sprintf("%s/messages?offset=0&offset=1&limit=10", ts.URL),
			token:  adminToken,
			status: http.StatusBadRequest,
		},
		{
			desc:   "read page with multiple limit",
			url:    fmt.Sprintf("%s/messages?offset=0&limit=20&limit=10", ts.URL),
			token:  adminToken,
			status: http.StatusBadRequest,
		},
		{
			desc:   "read page with empty token",
			url:    fmt.Sprintf("%s/messages?offset=0&limit=10", ts.URL),
			token:  "",
			status: http.StatusUnauthorized,
		},
		{
			desc:   "read page with default offset",
			url:    fmt.Sprintf("%s/messages?limit=10", ts.URL),
			token:  adminToken,
			status: http.StatusOK,
			res: pageRes{
				Total:    uint64(len(messages)),
				Messages: messages[0:10],
			},
		},
		{
			desc:   "read page with default limit",
			url:    fmt.Sprintf("%s/messages?offset=0", ts.URL),
			token:  adminToken,
			status: http.StatusOK,
			res: pageRes{
				Total:    uint64(len(messages)),
				Messages: messages[0:10],
			},
		},
		{
			desc:   "read page with senml format",
			url:    fmt.Sprintf("%s/messages?format=messages", ts.URL),
			token:  adminToken,
			status: http.StatusOK,
			res: pageRes{
				Total:    uint64(len(messages)),
				Messages: messages[0:10],
			},
		},
		{
			desc:   "read page with subtopic",
			url:    fmt.Sprintf("%s/messages?subtopic=%s&protocol=%s", ts.URL, subtopic, httpProt),
			token:  adminToken,
			status: http.StatusOK,
			res: pageRes{
				Total:    uint64(len(queryMsgs)),
				Messages: queryMsgs[0:10],
			},
		},
		{
			desc:   "read page with subtopic and protocol",
			url:    fmt.Sprintf("%s/messages?subtopic=%s&protocol=%s", ts.URL, subtopic, httpProt),
			token:  adminToken,
			status: http.StatusOK,
			res: pageRes{
				Total:    uint64(len(queryMsgs)),
				Messages: queryMsgs[0:10],
			},
		},
		{
			desc:   "read page with protocol",
			url:    fmt.Sprintf("%s/messages?protocol=http", ts.URL),
			token:  adminToken,
			status: http.StatusOK,
			res: pageRes{
				Total:    uint64(len(queryMsgs)),
				Messages: queryMsgs[0:10],
			},
		},
		{
			desc:   "read page with name",
			url:    fmt.Sprintf("%s/messages?name=%s", ts.URL, msgName),
			token:  adminToken,
			status: http.StatusOK,
			res: pageRes{
				Total:    uint64(len(queryMsgs)),
				Messages: queryMsgs[0:10],
			},
		},
		{
			desc:   "read page with value",
			url:    fmt.Sprintf("%s/messages?v=%f", ts.URL, v),
			token:  adminToken,
			status: http.StatusOK,
			res: pageRes{
				Total:    uint64(len(valueMsgs)),
				Messages: valueMsgs[0:10],
			},
		},
		{
			desc:   "read page with value and equal comparator",
			url:    fmt.Sprintf("%s/messages?v=%f&comparator=%s", ts.URL, v, readers.EqualKey),
			token:  adminToken,
			status: http.StatusOK,
			res: pageRes{
				Total:    uint64(len(valueMsgs)),
				Messages: valueMsgs[0:10],
			},
		},
		{
			desc:   "read page with value and lower-than comparator",
			url:    fmt.Sprintf("%s/messages?v=%f&comparator=%s", ts.URL, v+1, readers.LowerThanKey),
			token:  adminToken,
			status: http.StatusOK,
			res: pageRes{
				Total:    uint64(len(valueMsgs)),
				Messages: valueMsgs[0:10],
			},
		},
		{
			desc:   "read page with value and lower-than-or-equal comparator",
			url:    fmt.Sprintf("%s/messages?v=%f&comparator=%s", ts.URL, v+1, readers.LowerThanEqualKey),
			token:  adminToken,
			status: http.StatusOK,
			res: pageRes{
				Total:    uint64(len(valueMsgs)),
				Messages: valueMsgs[0:10],
			},
		},
		{
			desc:   "read page with value and greater-than comparator",
			url:    fmt.Sprintf("%s/messages?v=%f&comparator=%s", ts.URL, v-1, readers.GreaterThanKey),
			token:  adminToken,
			status: http.StatusOK,
			res: pageRes{
				Total:    uint64(len(valueMsgs)),
				Messages: valueMsgs[0:10],
			},
		},
		{
			desc:   "read page with value and greater-than-or-equal comparator",
			url:    fmt.Sprintf("%s/messages?v=%f&comparator=%s", ts.URL, v-1, readers.GreaterThanEqualKey),
			token:  adminToken,
			status: http.StatusOK,
			res: pageRes{
				Total:    uint64(len(valueMsgs)),
				Messages: valueMsgs[0:10],
			},
		},
		{
			desc:   "read page with non-float value",
			url:    fmt.Sprintf("%s/messages?v=ab01", ts.URL),
			token:  adminToken,
			status: http.StatusBadRequest,
		},
		{
			desc:   "read page with value and wrong comparator",
			url:    fmt.Sprintf("%s/messages?v=%f&comparator=wrong", ts.URL, v-1),
			token:  adminToken,
			status: http.StatusBadRequest,
		},
		{
			desc:   "read page with boolean value",
			url:    fmt.Sprintf("%s/messages?vb=true", ts.URL),
			token:  adminToken,
			status: http.StatusOK,
			res: pageRes{
				Total:    uint64(len(boolMsgs)),
				Messages: boolMsgs[0:10],
			},
		},
		{
			desc:   "read page with non-boolean value",
			url:    fmt.Sprintf("%s/messages?vb=yes", ts.URL),
			token:  adminToken,
			status: http.StatusBadRequest,
		},
		{
			desc:   "read page with string value",
			url:    fmt.Sprintf("%s/messages?vs=%s", ts.URL, vs),
			token:  adminToken,
			status: http.StatusOK,
			res: pageRes{
				Total:    uint64(len(stringMsgs)),
				Messages: stringMsgs[0:10],
			},
		},
		{
			desc:   "read page with data value",
			url:    fmt.Sprintf("%s/messages?vd=%s", ts.URL, vd),
			token:  adminToken,
			status: http.StatusOK,
			res: pageRes{
				Total:    uint64(len(dataMsgs)),
				Messages: dataMsgs[0:10],
			},
		},
		{
			desc:   "read page with non-float from",
			url:    fmt.Sprintf("%s/messages?from=ABCD", ts.URL),
			token:  adminToken,
			status: http.StatusBadRequest,
		},

		{
			desc:   "read page with non-float",
			url:    fmt.Sprintf("%s/messages?to=ABCD", ts.URL),
			token:  adminToken,
			status: http.StatusBadRequest,
		},
		{
			desc:   "read page with from/to ",
			url:    fmt.Sprintf("%s/messages?from=%f&to=%f", ts.URL, messages[19].Time, messages[4].Time),
			token:  adminToken,
			status: http.StatusOK,
			res: pageRes{
				Total:    uint64(len(messages[5:20])),
				Messages: messages[5:15],
			},
		},
	}

	for _, tc := range cases {
		req := testRequest{
			client: ts.Client(),
			method: http.MethodGet,
			url:    tc.url,
			token:  tc.token,
			key:    tc.key,
		}

		res, err := req.make()
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))

		var page pageRes
		json.NewDecoder(res.Body).Decode(&page)
		assert.Nil(t, err, fmt.Sprintf("%s: unexpected error %s", tc.desc, err))
		assert.Equal(t, tc.status, res.StatusCode, fmt.Sprintf("%s: expected %d got %d", tc.desc, tc.status, res.StatusCode))
		assert.Equal(t, tc.res.Total, page.Total, fmt.Sprintf("%s: expected %d got %d", tc.desc, tc.res.Total, page.Total))
		assert.ElementsMatch(t, tc.res.Messages, page.Messages, fmt.Sprintf("%s: expected body %v got %v", tc.desc, tc.res.Messages, page.Messages))
	}
}

type pageRes struct {
	readers.PageMetadata
	Total    uint64          `json:"total"`
	Messages []senml.Message `json:"messages,omitempty"`
}

func fromSenml(in []senml.Message) []readers.Message {
	var ret []readers.Message
	for _, m := range in {
		ret = append(ret, m)
	}
	return ret
}
