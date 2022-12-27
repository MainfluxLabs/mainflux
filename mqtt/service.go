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
	// ListSubscriptions lists subscriptions having the provided user token and search params.
	ListAllSubscriptions(ctx context.Context, token string, pm PageMetadata) (Page, error)
}

type mqttService struct {
	auth              mainflux.AuthServiceClient
	subscriptionsRepo Repository
}

// NewMqttService instantiates the MQTT service implementation.
func NewMqttService(auth mainflux.AuthServiceClient, subscriptionsRepo Repository) Service {
	return &mqttService{
		auth:              auth,
		subscriptionsRepo: subscriptionsRepo,
	}
}

func (ms *mqttService) ListAllSubscriptions(ctx context.Context, token string, pm PageMetadata) (Page, error) {
	res, err := ms.auth.Identify(ctx, &mainflux.Token{Value: token})
	if err != nil {
		return Page{}, errors.Wrap(errors.ErrAuthentication, err)
	}

	if err := ms.authorize(ctx, res.GetId(), authoritiesObject, memberRelationKey); err == nil {
		return Page{}, errors.Wrap(errors.ErrAuthentication, err)
	}

	page, err := ms.subscriptionsRepo.RetrieveAll(ctx, pm)
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
