package webhooks

import (
	"context"
	"encoding/json"
	"net/http"

	clientshttp "github.com/MainfluxLabs/mainflux/pkg/clients/http"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
)

var errForward = errors.New("failed to forward message")

type Forwarder interface {
	// Forward method is used to forward the received message to a certain url
	Forward(ctx context.Context, message protomfx.Message, wh Webhook) error
}

var _ Forwarder = (*forwarder)(nil)

type forwarder struct{}

func NewForwarder() Forwarder {
	return &forwarder{}
}

func (fw *forwarder) Forward(_ context.Context, msg protomfx.Message, wh Webhook) error {
	body, err := json.Marshal(msg.Payload)
	if err != nil {
		return err
	}

	_, err = clientshttp.SendRequest(http.MethodPost, wh.Url, body, wh.Headers)
	if err != nil {
		return errors.Wrap(errForward, err)
	}

	return nil
}
