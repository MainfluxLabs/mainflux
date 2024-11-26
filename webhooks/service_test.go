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
	"github.com/MainfluxLabs/mainflux/things"
	"github.com/MainfluxLabs/mainflux/webhooks"
	whMock "github.com/MainfluxLabs/mainflux/webhooks/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	token       = "admin@example.com"
	wrongValue  = "wrong-value"
	emptyValue  = ""
	groupID     = "574106f7-030e-4881-8ab0-151195c29f94"
	prefixID    = "fe6b4e92-cc98-425e-b0aa-"
	prefixName  = "test-webhook-"
	webhookName = "test-webhook"
	nameKey     = "name"
	ascKey      = "asc"
	descKey     = "desc"
)

var (
	headers  = map[string]string{"Content-Type:": "application/json"}
	metadata = map[string]interface{}{"test": "data"}
	webhook  = webhooks.Webhook{GroupID: groupID, Name: webhookName, Url: "https://test.webhook.com", Headers: headers, Metadata: metadata}
)

func newService() webhooks.Service {
	ths := mocks.NewThingsServiceClient(nil, nil, map[string]things.Group{token: {ID: groupID}})
	webhookRepo := whMock.NewWebhookRepository()
	forwarder := whMock.NewForwarder()
	idProvider := uuid.NewMock()

	return webhooks.New(ths, webhookRepo, forwarder, idProvider)
}

func TestCreateWebhooks(t *testing.T) {
	svc := newService()
	var whs []webhooks.Webhook
	for i := 0; i < 3; i++ {
		id := fmt.Sprintf("%s%012d", prefixID, i+1)
		name := fmt.Sprintf("%s%012d", prefixName, i+1)
		webhook1 := webhook
		webhook1.ID = id
		webhook1.Name = name
		whs = append(whs, webhook1)
	}

	invalidGroupWh := webhook
	invalidGroupWh.GroupID = emptyValue

	invalidNameWh := webhook
	invalidNameWh.Name = emptyValue

	invalidUrlWh := webhook
	invalidUrlWh.Url = wrongValue

	cases := []struct {
		desc     string
		webhooks []webhooks.Webhook
		token    string
		err      error
	}{
		{
			desc:     "create new webhooks",
			webhooks: whs,
			token:    token,
			err:      nil,
		},
		{
			desc:     "create webhook with wrong credentials",
			webhooks: whs,
			token:    wrongValue,
			err:      errors.ErrAuthentication,
		},
		{
			desc:     "create webhook with invalid group id",
			webhooks: []webhooks.Webhook{invalidGroupWh},
			token:    token,
			err:      errors.ErrAuthorization,
		},
		{
			desc:     "create webhook with invalid name",
			webhooks: []webhooks.Webhook{invalidNameWh},
			token:    token,
			err:      nil,
		},
		{
			desc:     "create webhook with invalid url",
			webhooks: []webhooks.Webhook{invalidUrlWh},
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
	var whs []webhooks.Webhook
	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("%s%012d", prefixID, i+1)
		name := fmt.Sprintf("%s%012d", prefixName, i+1)
		webhook1 := webhook
		webhook1.ID = id
		webhook1.Name = name
		whs = append(whs, webhook1)
	}
	whs, err := svc.CreateWebhooks(context.Background(), token, whs...)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))
	cases := []struct {
		desc         string
		token        string
		grID         string
		pageMetadata webhooks.PageMetadata
		size         uint64
		err          error
	}{
		{
			desc:  "list the webhooks by group",
			token: token,
			grID:  groupID,
			pageMetadata: webhooks.PageMetadata{
				Offset: 0,
				Limit:  uint64(len(whs)),
			},
			size: uint64(len(whs)),
			err:  nil,
		},
		{
			desc:  "list the webhooks by group with no limit",
			token: token,
			grID:  groupID,
			pageMetadata: webhooks.PageMetadata{
				Limit: 0,
			},
			size: uint64(len(whs)),
			err:  nil,
		},
		{
			desc:  "list last webhook by group",
			token: token,
			grID:  groupID,
			pageMetadata: webhooks.PageMetadata{
				Offset: uint64(len(whs)) - 1,
				Limit:  uint64(len(whs)),
			},
			size: 1,
			err:  nil,
		},
		{
			desc:  "list empty set of webhooks by group",
			token: token,
			grID:  groupID,
			pageMetadata: webhooks.PageMetadata{
				Offset: uint64(len(whs)) + 1,
				Limit:  uint64(len(whs)),
			},
			size: 0,
			err:  nil,
		},
		{
			desc:  "list webhooks with invalid auth token",
			token: wrongValue,
			grID:  groupID,
			pageMetadata: webhooks.PageMetadata{
				Offset: 0,
				Limit:  0,
			},
			size: 0,
			err:  errors.ErrAuthentication,
		},
		{
			desc:  "list webhooks with invalid group id",
			token: token,
			grID:  emptyValue,
			pageMetadata: webhooks.PageMetadata{
				Offset: 0,
				Limit:  0,
			},
			size: 0,
			err:  errors.ErrAuthorization,
		},
		{
			desc:  "list webhooks by group sorted by name ascendant",
			token: token,
			grID:  groupID,
			pageMetadata: webhooks.PageMetadata{
				Offset: 0,
				Limit:  uint64(len(whs)),
				Order:  nameKey,
				Dir:    ascKey,
			},
			size: uint64(len(whs)),
			err:  nil,
		},
		{
			desc:  "list webhooks by group sorted by name descendent",
			token: token,
			grID:  groupID,
			pageMetadata: webhooks.PageMetadata{
				Offset: 0,
				Limit:  uint64(len(whs)),
				Order:  nameKey,
				Dir:    descKey,
			},
			size: uint64(len(whs)),
			err:  nil,
		},
	}

	for desc, tc := range cases {
		page, err := svc.ListWebhooksByGroup(context.Background(), tc.token, tc.grID, tc.pageMetadata)
		size := uint64(len(page.Webhooks))
		assert.Equal(t, tc.size, size, fmt.Sprintf("%v: expected %v got %v\n", desc, tc.size, size))
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
			err:     errors.ErrAuthentication,
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
			err:   errors.ErrAuthentication,
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
			desc:  "remove webhook with wrong credentials",
			id:    wh.ID,
			token: wrongValue,
			err:   errors.ErrAuthentication,
		},
		{
			desc:  "remove existing webhook",
			id:    wh.ID,
			token: token,
			err:   nil,
		},
		{
			desc:  "remove non-existing webhook",
			id:    wrongValue,
			token: token,
			err:   errors.ErrNotFound,
		},
	}

	for _, tc := range cases {
		err := svc.RemoveWebhooks(context.Background(), tc.token, tc.id)
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
			ProfileConfig: map[string]interface{}{
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
			ProfileConfig: map[string]interface{}{
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
