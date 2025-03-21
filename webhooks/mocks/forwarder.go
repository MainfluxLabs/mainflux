package mocks

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/transformers/json"
	"github.com/MainfluxLabs/mainflux/webhooks"
)

var _ webhooks.Forwarder = (*forwarder)(nil)

type forwarder struct{}

func NewForwarder() webhooks.Forwarder {
	return &forwarder{}
}

func (mf *forwarder) Forward(ctx context.Context, message json.Message, wh webhooks.Webhook) error {
	return nil
}
