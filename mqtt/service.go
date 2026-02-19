package mqtt

import (
	"context"

	"github.com/MainfluxLabs/mainflux/mqtt/redis/cache"
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

	// RemoveSubscriptionsByThing removes all subscriptions associated with the specified thing ID.
	RemoveSubscriptionsByThing(ctx context.Context, thingID string) error

	// RemoveSubscriptionsByGroup removes all subscriptions associated with the specified group ID.
	RemoveSubscriptionsByGroup(ctx context.Context, groupID string) error
}

type mqttService struct {
	auth          protomfx.AuthServiceClient
	things        protomfx.ThingsServiceClient
	subscriptions Repository
	cache         cache.ConnectionCache
	idp           uuid.IDProvider
}

// NewMqttService instantiates the MQTT service implementation.
func NewMqttService(auth protomfx.AuthServiceClient, things protomfx.ThingsServiceClient, subscriptions Repository, cache cache.ConnectionCache, idp uuid.IDProvider) Service {
	return &mqttService{
		auth:          auth,
		things:        things,
		subscriptions: subscriptions,
		cache:         cache,
		idp:           idp,
	}
}

func (ms *mqttService) CreateSubscription(ctx context.Context, sub Subscription) error {
	return ms.subscriptions.Save(ctx, sub)
}

func (ms *mqttService) ListSubscriptions(ctx context.Context, groupID, token string, pm PageMetadata) (Page, error) {
	if _, err := ms.things.CanUserAccessGroup(ctx, &protomfx.UserAccessReq{Token: token, Id: groupID, Action: things.Viewer}); err != nil {
		return Page{}, err
	}

	return ms.subscriptions.RetrieveByGroup(ctx, pm, groupID)
}

func (ms *mqttService) RemoveSubscription(ctx context.Context, sub Subscription) error {
	return ms.subscriptions.Remove(ctx, sub)
}

func (ms *mqttService) RemoveSubscriptionsByThing(ctx context.Context, thingID string) error {
	if err := ms.subscriptions.RemoveByThing(ctx, thingID); err != nil {
		return err
	}

	// Disconnect all active MQTT clients for the given thing.
	// This is done only when removing subscriptions by thing/group,
	// since single subscription removal must not terminate active connections.
	if err := ms.cache.DisconnectByThing(ctx, thingID); err != nil {
		return err
	}

	return nil
}

func (ms *mqttService) RemoveSubscriptionsByGroup(ctx context.Context, groupID string) error {
	page, err := ms.subscriptions.RetrieveByGroup(ctx, PageMetadata{}, groupID)
	if err != nil {
		return err
	}

	if err = ms.subscriptions.RemoveByGroup(ctx, groupID); err != nil {
		return err
	}

	disconnected := map[string]bool{}
	for _, sub := range page.Subscriptions {
		if disconnected[sub.ThingID] {
			continue
		}
		if err := ms.cache.DisconnectByThing(ctx, sub.ThingID); err != nil {
			return err
		}
		disconnected[sub.ThingID] = true
	}

	return nil
}
