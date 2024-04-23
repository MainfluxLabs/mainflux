package webhooks

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	mfjson "github.com/MainfluxLabs/mainflux/pkg/transformers/json"
)

type Forwarder interface {
	// Forward method is used to forward the received message to a certain url
	Forward(ctx context.Context, message mfjson.Message, whs []Webhook) error
}

var _ Forwarder = (*forwarder)(nil)

type forwarder struct {
	httpClient *http.Client
}

func NewForwarder() Forwarder {
	return &forwarder{
		httpClient: &http.Client{},
	}
}

func (fw *forwarder) Forward(_ context.Context, msg mfjson.Message, whs []Webhook) error {
	for _, wh := range whs {
		if err := fw.sendRequest(wh.Url, msg); err != nil {
			return err
		}
	}

	return nil
}

func (fw *forwarder) sendRequest(url string, msg mfjson.Message) error {
	jsonBytes, err := json.Marshal(msg.Payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(jsonBytes))
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
