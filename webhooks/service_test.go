package webhooks_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/MainfluxLabs/mainflux/internal/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/pkg/mocks"
	"github.com/MainfluxLabs/mainflux/webhooks"
	whMock "github.com/MainfluxLabs/mainflux/webhooks/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	thingID    = "1"
	token      = "admin@example.com"
	wrongValue = "wrong-value"
	emptyValue = ""
)

func newService() webhooks.Service {
	things := mocks.NewThingsServiceClient(nil, map[string]string{token: thingID}, nil)
	webhookRepo := whMock.NewWebhookRepository()

	return webhooks.New(things, webhookRepo)
}

func TestCreateWebhooks(t *testing.T) {
	svc := newService()

	validDataWebhooks := []webhooks.Webhook{{ThingID: "1", Name: "test1", Format: "json", Url: "http://test1.com"}, {ThingID: "1", Name: "test2", Format: "json", Url: "https://test2.com"}}
	validDataWebhook := []webhooks.Webhook{{ThingID: "1", Name: "test3", Format: "json", Url: "http://test3.com"}}
	invalidThingData := []webhooks.Webhook{{ThingID: emptyValue, Name: "test4", Format: "json", Url: "http://test4.com"}}
	invalidNameData := []webhooks.Webhook{{ThingID: "1", Name: emptyValue, Format: "json", Url: "https://test3.com"}}
	invalidFormatData := []webhooks.Webhook{{ThingID: "1", Name: "test5", Format: emptyValue, Url: "https://test3.com"}}
	invalidUrlData := []webhooks.Webhook{{ThingID: "1", Name: "test6", Format: "json", Url: emptyValue}}

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
			webhooks: validDataWebhook,
			token:    wrongValue,
			err:      errors.ErrAuthorization,
		},
		{
			desc:     "create webhook with invalid thing id",
			webhooks: invalidThingData,
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
			desc:     "create webhook with invalid format",
			webhooks: invalidFormatData,
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

func TestListWebhooksByThing(t *testing.T) {
	svc := newService()

	w := webhooks.Webhook{
		Name:    "test-webhook",
		ThingID: "1",
		Format:  "json",
		Url:     "https://api.webhook.com",
	}

	whs, err := svc.CreateWebhooks(context.Background(), token, w)
	require.Nil(t, err, fmt.Sprintf("unexpected error: %s", err))

	cases := []struct {
		desc     string
		webhooks []webhooks.Webhook
		thID     string
		token    string
		err      error
	}{
		{
			desc:     "list the webhooks",
			webhooks: whs,
			thID:     thingID,
			token:    token,
			err:      nil,
		},
		{
			desc:     "list webhooks with invalid auth token",
			webhooks: []webhooks.Webhook{},
			thID:     thingID,
			token:    wrongValue,
			err:      errors.ErrAuthorization,
		},
		{
			desc:     "list webhooks with invalid thing id",
			webhooks: []webhooks.Webhook{},
			thID:     wrongValue,
			token:    token,
			err:      errors.ErrAuthorization,
		},
	}

	for desc, tc := range cases {
		whs, err := svc.ListWebhooksByThing(context.Background(), tc.token, tc.thID)
		assert.Equal(t, tc.webhooks, whs, fmt.Sprintf("%v: expected %v got %v\n", desc, tc.webhooks, whs))
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%v: expected %s got %s\n", desc, tc.err, err))
	}
}

func TestConsume(t *testing.T) {
	svc := newService()

	validData := messaging.Message{Publisher: thingID, Profile: &messaging.Profile{Webhook: true}, Payload: []byte(`{"api_key":"val1","field1":"val2","field2":"val3","field3":"val4"}`)}
	invalidThingData := messaging.Message{Publisher: emptyValue, Profile: &messaging.Profile{Webhook: true}, Payload: []byte(`{"api_key":"val1","field1":"val2","field2":"val3","field3":"val4"}`)}

	cases := []struct {
		desc string
		msg  messaging.Message
		err  error
	}{
		{
			desc: "forward message",
			msg:  validData,
			err:  nil,
		},
		{
			desc: "forward message invalid thing id",
			msg:  invalidThingData,
			err:  apiutil.ErrMissingID,
		},
	}

	for _, tc := range cases {
		err := svc.Consume(tc.msg)
		assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
	}
}
