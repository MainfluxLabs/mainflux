package mocks

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/errors"
	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/webhooks"
)

// InvalidURL is a webhook URL for which the mock forwarder always fails.
const InvalidURL = "https://invalid.webhook.com"

var (
	_ webhooks.Forwarder = (*forwarder)(nil)

	errForward = errors.New("failed to forward message")
)

type forwarder struct{}

func NewForwarder() webhooks.Forwarder {
	return &forwarder{}
}

func (mf *forwarder) Forward(_ context.Context, _ protomfx.Webhook, wh webhooks.Webhook) error {
	if wh.Url == InvalidURL {
		return errForward
	}

	return nil
}
