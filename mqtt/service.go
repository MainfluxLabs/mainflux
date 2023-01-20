package mqtt

import (
	"context"

	"github.com/MainfluxLabs/mainflux"
	"github.com/MainfluxLabs/mainflux/pkg/errors"
)

const (
	usersObjectKey    = "users"
	authoritiesObject = "authorities"
	memberRelationKey = "member"
)

// Service specifies an API that must be fullfiled by the domain service
// implementation, and all of its decorators (e.g. logging & metrics).
type Service interface {
	// ListSubscriptions lists all subscriptions that belong to the specified owner.
	ListSubscriptions(ctx context.Context, token string, pm PageMetadata) (Page, error)
	// CreateSubscription create a subscription.
	CreateSubscription(ctx context.Context, token string, sub Subscription) error
	// RemoveSubscription removes the subscription having the provided identifier.
	RemoveSubscription(ctx context.Context, token string, sub Subscription) error
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

func (ms *mqttService) CreateSubscription(ctx context.Context, token string, sub Subscription) error {
	res, err := ms.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return errors.Wrap(errors.ErrAuthentication, err)
	}

	sub.OwnerID = res.GetId()
	if err != nil {
		return err
	}

	return ms.subscriptions.Save(ctx, sub)

}

func (ms *mqttService) RemoveSubscription(ctx context.Context, token string, sub Subscription) error {
	_, err := ms.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return errors.Wrap(errors.ErrAuthentication, err)
	}

	err = ms.subscriptions.Remove(ctx, sub)
	if err != nil {
		return err
	}

	return nil
}

func (ms *mqttService) ListSubscriptions(ctx context.Context, token string, pm PageMetadata) (Page, error) {
	res, err := ms.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Page{}, errors.Wrap(errors.ErrAuthentication, err)
	}

	if err := ms.authorize(ctx, res.GetId(), authoritiesObject, memberRelationKey); err == nil {
		return Page{}, errors.Wrap(errors.ErrAuthentication, err)
	}

	ownerID := res.GetId()
	page, err := ms.subscriptions.RetrieveByOwnerID(ctx, pm, ownerID)
	if err != nil {
		return Page{}, err
	}

	return page, nil
}

func (svc mqttService) authorize(ctx context.Context, subject, object, relation string) error {
	req := &mainflux.AuthorizeReq{
		Sub: subject,
		Obj: object,
		Act: relation,
	}
	res, err := svc.auth.Authorize(ctx, req)
	if err != nil {
		return errors.Wrap(errors.ErrAuthorization, err)
	}
	if !res.GetAuthorized() {
		return errors.ErrAuthorization
	}
	return nil
}
