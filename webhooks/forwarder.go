package webhooks

import (
	"bytes"
	"context"
	"net/http"

	"github.com/MainfluxLabs/mainflux/internal/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
)

type Forwarder interface {
	// Forward method is used to forward the received message to a certain url
	Forward(ctx context.Context, message messaging.Message) error
}

var _ Forwarder = (*forwarder)(nil)

type forwarder struct {
	webhooks   WebhookRepository
	httpClient *http.Client
}

func NewForwarder(webhooks WebhookRepository) Forwarder {
	return &forwarder{
		webhooks:   webhooks,
		httpClient: &http.Client{},
	}
}

func (fw *forwarder) Forward(ctx context.Context, msg messaging.Message) error {
	if msg.Publisher == "" {
		return apiutil.ErrMissingID
	}

	whs, err := fw.webhooks.RetrieveByThingID(ctx, msg.Publisher)
	if err != nil {
		return errors.ErrAuthorization
	}

	for _, wh := range whs {
		if err := fw.sendRequest(wh.Url, msg); err != nil {
			return err
		}
	}

	return nil
}

func (fw *forwarder) sendRequest(url string, msg messaging.Message) error {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(msg.Payload))
	if err != nil {
		return err
	}

	req.Header.Set(contentType, ctJSON)

	resp, err := fw.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(ErrSendRequest, err)
	}
	defer resp.Body.Close()

	return nil
}
