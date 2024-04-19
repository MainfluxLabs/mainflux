package webhooks

import (
	"bytes"
	"context"
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
)

type Forwarder interface {
	// Forward method is used to forward the received message to a certain url
	Forward(ctx context.Context, message messaging.Message, whs []Webhook) error
}

var _ Forwarder = (*forwarder)(nil)

type forwarder struct {
	httpClient *http.Client
}

func NewForwarder(webhooks WebhookRepository) Forwarder {
	return &forwarder{
		httpClient: &http.Client{},
	}
}

func (fw *forwarder) Forward(_ context.Context, msg messaging.Message, whs []Webhook) error {
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
