package webhooks_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/webhooks"
	whMock "github.com/MainfluxLabs/mainflux/webhooks/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	token      = "admin@example.com"
	wrongValue = "wrong-value"
	emptyValue = ""
	groupID    = "1"
)

var (
	headers = map[string]string{"Content-Type:": "application/json"}
	webhook = webhooks.Webhook{GroupID: groupID, Name: "test-webhook", Url: "https://test.webhook.com", Headers: headers}
)

func newService() webhooks.Service {
	things := mocks.NewThingsServiceClient(nil, map[string]string{token: groupID}, nil)
	webhookRepo := whMock.NewWebhookRepository()
	forwarder := whMock.NewForwarder()
	idProvider := uuid.NewMock()

	return webhooks.New(things, webhookRepo, forwarder, idProvider)
}

func TestCreateWebhooks(t *testing.T) {
	svc := newService()

	validData := webhooks.Webhook{GroupID: groupID, Name: "test1", Url: "http://test1.com", Headers: headers}
	validData2 := webhooks.Webhook{GroupID: groupID, Name: "test2", Url: "http://test2.com", Headers: headers}
	validDataWebhooks := []webhooks.Webhook{validData, validData2}
	invalidGroupData := []webhooks.Webhook{{GroupID: emptyValue, Name: "test3", Url: "http://test3.com", Headers: headers}}
	invalidNameData := []webhooks.Webhook{{GroupID: groupID, Name: emptyValue, Url: "https://test.com", Headers: headers}}
	invalidUrlData := []webhooks.Webhook{{GroupID: groupID, Name: "test5", Url: emptyValue, Headers: headers}}

	cases := []struct {
		desc     string
		webhooks []webhooks.Webhook
		token    string
		err      error
	}{
		{
			desc:     "create new webhooks",
			webhooks: validDataWebhooks,
			token:    token,
			err:      nil,
		},
		{
			desc:     "create webhook with wrong credentials",
			webhooks: []webhooks.Webhook{validData},
			token:    wrongValue,
			err:      errors.ErrAuthorization,
		},
		{
			desc:     "create webhook with invalid group id",
			webhooks: invalidGroupData,
			token:    token,
			err:      errors.ErrAuthorization,
		},
		{
			desc:     "create webhook with invalid name",
			webhooks: invalidNameData,
			token:    token,
			err:      nil,
		},
		{
			desc:     "create webhook with invalid url",
			webhooks: invalidUrlData,
			token:    token,
			err:      nil,
		},
	}

	for desc, tc := range cases {
		_, err := svc.CreateWebhooks(context.Background(), tc.token, tc.webhooks...)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%v: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestListWebhooksByGroup(t *testing.T) {
	svc := newService()
	whs, err := svc.CreateWebhooks(context.Background(), token, webhook)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc     string
		webhooks []webhooks.Webhook
		token    string
		grID     string
		err      error
	}{
		{
			desc:     "list the webhooks",
			webhooks: whs,
			token:    token,
			grID:     groupID,
			err:      nil,
		},
		{
			desc:     "list webhooks with invalid auth token",
			webhooks: []webhooks.Webhook{},
			token:    wrongValue,
			grID:     groupID,
			err:      errors.ErrAuthorization,
		},
		{
			desc:     "list webhooks with invalid group id",
			webhooks: []webhooks.Webhook{},
			token:    token,
			err:      errors.ErrAuthorization,
		},
	}

	for desc, tc := range cases {
		whs, err := svc.ListWebhooksByGroup(context.Background(), tc.token, tc.grID)
		assert.Equal(t, tc.webhooks, whs, fmt.Sprintf("%v: expected %v got %v\n", desc, tc.webhooks, whs))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%v: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestUpdateWebhook(t *testing.T) {
	svc := newService()
	whs, err := svc.CreateWebhooks(context.Background(), token, webhook)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	wh := whs[0]

	invalidWh := webhooks.Webhook{ID: emptyValue, Name: wh.Name, Url: wh.Url, GroupID: wh.GroupID, Headers: wh.Headers}

	cases := []struct {
		desc    string
		webhook webhooks.Webhook
		token   string
		err     error
	}{
		{
			desc:    "update existing webhook",
			webhook: wh,
			token:   token,
			err:     nil,
		},
		{
			desc:    "update webhook with wrong credentials",
			webhook: wh,
			token:   emptyValue,
			err:     errors.ErrAuthorization,
		},
		{
			desc:    "update non-existing webhook",
			webhook: invalidWh,
			token:   token,
			err:     errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.UpdateWebhook(context.Background(), tc.token, tc.webhook)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestViewWebhook(t *testing.T) {
	svc := newService()
	whs, err := svc.CreateWebhooks(context.Background(), token, webhook)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	wh := whs[0]

	cases := map[string]struct {
		id    string
		token string
		err   error
	}{
		"view existing webhook": {
			id:    wh.ID,
			token: token,
			err:   nil,
		},
		"view webhook with wrong credentials": {
			id:    wh.ID,
			token: wrongValue,
			err:   errors.ErrAuthorization,
		},
		"view non-existing webhook": {
			id:    wrongValue,
			token: token,
			err:   errors.ErrNotFound,
		},
	}

	for desc, tc := range cases {
		_, err := svc.ViewWebhook(context.Background(), tc.token, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestRemoveWebhooks(t *testing.T) {
	svc := newService()
	whs, err := svc.CreateWebhooks(context.Background(), token, webhook)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	wh := whs[0]

	cases := []struct {
		desc  string
		id    string
		token string
		err   error
	}{
		{
			desc:  "remove existing webhook",
			id:    wh.ID,
			token: token,
			err:   nil,
		},
		{
			desc:  "remove webhook with wrong credentials",
			id:    wh.ID,
			token: wrongValue,
			err:   errors.ErrAuthorization,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveWebhooks(context.Background(), tc.token, groupID, tc.id)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}

func TestConsume(t *testing.T) {
	svc := newService()
	whs, err := svc.CreateWebhooks(context.Background(), token, webhook)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	wh := whs[0]

	validJson := json.Messages{
		Data: []json.Message{{
			Profile: map[string]interface{}{
				"webhook_id": wh.ID,
			},
			Payload: map[string]interface{}{
				"key1": "val1",
				"key2": float64(123),
			}},
		},
	}

	withoutWh := json.Messages{
		Data: []json.Message{{
			Payload: map[string]interface{}{
				"key1": "val1",
				"key2": float64(123),
			}},
		},
	}

	emptyWh := json.Messages{
		Data: []json.Message{{
			Profile: map[string]interface{}{
				"webhook_id": emptyValue,
			},
			Payload: map[string]interface{}{
				"key1": "val1",
				"key2": float64(123),
			}},
		},
	}

	cases := []struct {
		desc string
		msg  json.Messages
		err  error
	}{
		{
			desc: "forward message",
			msg:  validJson,
			err:  nil,
		},
		{
			desc: "forward message without webhook id",
			msg:  withoutWh,
			err:  apiutil.ErrMissingID,
		},
		{
			desc: "forward message with empty webhook id",
			msg:  emptyWh,
			err:  errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.Consume(tc.msg)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
