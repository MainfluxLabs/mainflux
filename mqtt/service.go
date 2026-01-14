package mqtt

import (
	"context"

	protomfx "github.com/MainfluxLabs/mainflux/pkg/proto"
	"github.com/MainfluxLabs/mainflux/pkg/uuid"
	"github.com/MainfluxLabs/mainflux/things"
)

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// ListSubscriptions lists all subscriptions that belong to the specified group.
	ListSubscriptions(ctx context.Context, groupID, token string, pm PageMetadata) (Page, error)

	// CreateSubscription create a subscription.
	CreateSubscription(ctx context.Context, sub Subscription) error

	// RemoveSubscription removes the subscription having the provided identifier.
	RemoveSubscription(ctx context.Context, sub Subscription) error
}

type mqttService struct {
	auth          protomfx.AuthServiceClient
	things        protomfx.ThingsServiceClient
	subscriptions Repository
	idp           uuid.IDProvider
}

// NewMqttService instantiates the MQTT service implementation.
func NewMqttService(auth protomfx.AuthServiceClient, things protomfx.ThingsServiceClient, subscriptions Repository, idp uuid.IDProvider) Service {
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
	return ms.subscriptions.Remove(ctx, sub)
}

func (ms *mqttService) ListSubscriptions(ctx context.Context, groupID, token string, pm PageMetadata) (Page, error) {
	if _, err := ms.things.CanUserAccessGroup(ctx, &protomfx.UserAccessReq{Token: token, Id: groupID, Action: things.Viewer}); err != nil {
		return Page{}, err
	}

	return ms.subscriptions.RetrieveByGroup(ctx, pm, groupID)
}
