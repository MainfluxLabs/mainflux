package mqtt

import (
	"context"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/auth"
)

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// ListSubscriptions lists all subscriptions that belong to the specified channel.
	ListSubscriptions(ctx context.Context, chanID, token, key string, pm PageMetadata) (Page, error)

	// CreateSubscription create a subscription.
	CreateSubscription(ctx context.Context, sub Subscription) error

	// RemoveSubscription removes the subscription having the provided identifier.
	RemoveSubscription(ctx context.Context, sub Subscription) error

	// HasClientID  indicates if a subscription exist for a given client ID.
	HasClientID(ctx context.Context, clientID string) error

	// UpdateStatus updates the subscription status for a given client ID.
	UpdateStatus(ctx context.Context, sub Subscription) error
}

type mqttService struct {
	auth          mainflux.AuthServiceClient
	things        mainflux.ThingsServiceClient
	subscriptions Repository
	idp           mainflux.IDProvider
}

// NewMqttService instantiates the MQTT service implementation.
func NewMqttService(auth mainflux.AuthServiceClient, things mainflux.ThingsServiceClient, subscriptions Repository, idp mainflux.IDProvider) Service {
	return &mqttService{
		auth:          auth,
		things:        things,
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

func (ms *mqttService) ListSubscriptions(ctx context.Context, chanID, token, key string, pm PageMetadata) (Page, error) {
	if err := ms.authorize(ctx, token, key, chanID); err != nil {
		return Page{}, err
	}

	return ms.subscriptions.RetrieveByChannelID(ctx, pm, chanID)
}

func (ms *mqttService) UpdateStatus(ctx context.Context, sub Subscription) error {
	return ms.subscriptions.UpdateStatus(ctx, sub)
}

func (ms *mqttService) HasClientID(ctx context.Context, clientID string) error {
	return ms.subscriptions.HasClientID(ctx, clientID)
}

func (ms *mqttService) authorize(ctx context.Context, token, key, chanID string) (err error) {
	switch {
	case token != "":
		if _, err := ms.auth.Authorize(ctx, &mainflux.AuthorizeReq{Token: token, Subject: auth.RootSubject}); err == nil {
			return nil
		}

		if _, err = ms.things.IsChannelOwner(ctx, &mainflux.ChannelOwnerReq{Token: token, ChanID: chanID}); err != nil {
			return err
		}
		return nil
	default:
		if _, err := ms.things.GetConnByKey(ctx, &mainflux.ConnByKeyReq{Key: key}); err != nil {
			return err
		}
		return nil
	}
}
