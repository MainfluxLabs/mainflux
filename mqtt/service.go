package mqtt

import (
	"context"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// ListSubscriptions lists all subscriptions that belong to the specified channel.
	ListSubscriptions(ctx context.Context, chanID, token string, pm PageMetadata) (Page, error)
	// CreateSubscription create a subscription.
	CreateSubscription(ctx context.Context, sub Subscription) error
	// RemoveSubscription removes the subscription having the provided identifier.
	RemoveSubscription(ctx context.Context, sub Subscription) error
}

type mqttService struct {
	auth          mainflux.AuthServiceClient
	subscriptions Repository
	idp           mainflux.IDProvider
}

// NewMqttService instantiates the MQTT service implementation.
func NewMqttService(auth mainflux.AuthServiceClient, subscriptions Repository, idp mainflux.IDProvider) Service {
	return &mqttService{
		auth:          auth,
		subscriptions: subscriptions,
		idp:           idp,
	}
}

func (ms *mqttService) CreateSubscription(ctx context.Context, sub Subscription) error {
	return ms.subscriptions.Save(ctx, sub)
}

func (ms *mqttService) RemoveSubscription(ctx context.Context, sub Subscription) error {
	err := ms.subscriptions.Remove(ctx, sub)
	if err != nil {
		return err
	}

	return nil
}

func (ms *mqttService) ListSubscriptions(ctx context.Context, chanID, token string, pm PageMetadata) (Page, error) {
	_, err := ms.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Page{}, errors.Wrap(errors.ErrAuthentication, err)
	}

	page, err := ms.subscriptions.RetrieveByChannelID(ctx, pm, chanID)
	if err != nil {
		return Page{}, err
	}

	return page, nil
}
