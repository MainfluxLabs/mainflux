package webhooks_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/internal/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	things2 "github.com/MainfluxLabs/mainflux/things"
	"github.com/MainfluxLabs/mainflux/webhooks"
	whMock "github.com/MainfluxLabs/mainflux/webhooks/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	groupID     = "testID"
	token       = "admin@example.com"
	wrongValue  = "wrong-value"
	emptyValue  = ""
	ownerID     = "ownerID"
	orgID       = "orgID"
	name        = "name"
	description = "description"
)

func newService() webhooks.Service {
	things := mocks.NewThingsServiceClient(nil, nil, createGroup())
	webhookRepo := whMock.NewWebhookRepository()
	forwarder := whMock.NewForwarder()
	idProvider := uuid.NewMock()

	return webhooks.New(things, webhookRepo, forwarder, idProvider)
}

func createGroup() map[string]things2.Group {
	group := map[string]things2.Group{}

	group[groupID] = things2.Group{
		ID:          groupID,
		OwnerID:     ownerID,
		OrgID:       orgID,
		Name:        name,
		Description: description,
		Metadata:    map[string]interface{}{"meta": "data"},
	}

	return group
}

func TestCreateWebhooks(t *testing.T) {
	svc := newService()

	validData := webhooks.Webhook{GroupID: "1", Name: "test1", Url: "http://test1.com", Headers: "test1"}
	validData2 := webhooks.Webhook{GroupID: "1", Name: "test2", Url: "http://test2.com", Headers: "test2"}
	validDataWebhooks := []webhooks.Webhook{validData, validData2}
	invalidGroupData := []webhooks.Webhook{{GroupID: emptyValue, Name: "test3", Url: "http://test3.com", Headers: "test1"}}
	invalidNameData := []webhooks.Webhook{{GroupID: "1", Name: emptyValue, Url: "https://test.com", Headers: "test1"}}
	invalidUrlData := []webhooks.Webhook{{GroupID: "1", Name: "test5", Url: emptyValue, Headers: "test1"}}

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

	w := webhooks.Webhook{
		Name:    "TestWebhook",
		GroupID: "1",
		Url:     "https://api.webhook.com",
		Headers: "",
	}

	whs, err := svc.CreateWebhooks(context.Background(), token, w)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc     string
		webhooks []webhooks.Webhook
		grID     string
		token    string
		err      error
	}{
		{
			desc:     "list the webhooks",
			webhooks: whs,
			grID:     groupID,
			token:    token,
			err:      nil,
		},
		{
			desc:     "list webhooks with invalid auth token",
			webhooks: []webhooks.Webhook{},
			grID:     groupID,
			token:    wrongValue,
			err:      errors.ErrAuthorization,
		},
		{
			desc:     "list webhooks with invalid group id",
			webhooks: []webhooks.Webhook{},
			grID:     wrongValue,
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

func TestConsume(t *testing.T) {
	svc := newService()

	validJson := json.Messages{
		Data: []json.Message{{
			Channel: "1",
			Payload: map[string]interface{}{
				"key1": "val1",
				"key2": float64(123),
			}},
		},
	}

	invalidJson := json.Messages{
		Data: []json.Message{{
			Channel: emptyValue,
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
			desc: "forward message with invalid channel id",
			msg:  invalidJson,
			err:  apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		err := svc.Consume(tc.msg)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
