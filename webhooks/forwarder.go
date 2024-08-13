package webhooks

import (
	"context"
	"net/http"

	clientshttp "github.com/MainfluxLabs/mainflux/pkg/clients/http"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
	mfjson "github.com/MainfluxLabs/mainflux/pkg/transformers/json"
)

type Forwarder interface {
	// Forward method is used to forward the received message to a certain url
	Forward(ctx context.Context, message mfjson.Message, wh Webhook) error
}

var _ Forwarder = (*forwarder)(nil)

type forwarder struct{}

func NewForwarder() Forwarder {
	return &forwarder{}
}

func (fw *forwarder) Forward(_ context.Context, msg mfjson.Message, wh Webhook) error {
	_, err := clientshttp.SendRequest(http.MethodPost, wh.Url, msg.Payload, wh.Headers)
	if err != nil {
		return errors.Wrap(clientshttp.ErrSendRequest, err)
	}

	return nil
}
