package mocks

import (
	"context"

	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/webhooks"
)

var _ webhooks.Forwarder = (*forwarder)(nil)

type forwarder struct{}

func NewForwarder() webhooks.Forwarder {
	return &forwarder{}
}

func (mf *forwarder) Forward(context.Context, protomfx.Message, webhooks.Webhook) error {
	return nil
}
