package mocks

import (
	"context"

	"github.com/MainfluxLabs/mainflux/internal/apiutil"
	"github.com/MainfluxLabs/mainflux/pkg/messaging"
	"github.com/MainfluxLabs/mainflux/webhooks"
)

var _ webhooks.Forwarder = (*forwarder)(nil)

type forwarder struct{}

func NewForwarder() webhooks.Forwarder {
	return &forwarder{}
}

func (mf *forwarder) Forward(ctx context.Context, message messaging.Message) error {
	if message.Publisher == "" {
		return apiutil.ErrMissingID
	}

	return nil
}
