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
	payloadCT := msg.Profile.ContentType
	var formattedPayload []byte
	var err error

	for _, wh := range whs {
		switch payloadCT {
		case ctJSON:
			formattedPayload, err = wh.Formatter.FormatJSONPayload(msg.Payload, wh.Formatter.Fields)
			if err != nil {
				return err
			}
		case senmlJson, senmlCbor:
			formattedPayload, err = wh.Formatter.FormatSenMLPayload(msg.Payload, wh.Formatter.Fields, payloadCT)
			if err != nil {
				return err
			}
		default:
			return apiutil.ErrUnsupportedContentType
		}

		if err := fw.sendRequest(wh.Url, formattedPayload); err != nil {
			return err
		}
	}

	return nil
}

func (fw *forwarder) sendRequest(url string, payload []byte) error {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
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
