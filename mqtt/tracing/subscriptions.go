package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/mqtt"
	"github.com/MainfluxLabs/mainflux/pkg/dbutil"
	"github.com/opentracing/opentracing-go"
)

const (
	saveSubscription            = "save_subscription"
	retrieveSubscriptionByGroup = "retrieve_subscription_by_group"
	removeSubscription          = "remove_subscription"
	removeSubscriptionByThing   = "remove_subscription_by_thing"
	removeSubscriptionByGroup   = "remove_subscription_by_group"
)

var _ mqtt.Repository = (*subscriptionRepositoryMiddleware)(nil)

type subscriptionRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   mqtt.Repository
}

// SubscriptionRepositoryMiddleware tracks request and their latency, and adds spans to context.
func SubscriptionRepositoryMiddleware(tracer opentracing.Tracer, repo mqtt.Repository) mqtt.Repository {
	return subscriptionRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (srm subscriptionRepositoryMiddleware) Save(ctx context.Context, sub mqtt.Subscription) error {
	span := dbutil.CreateSpan(ctx, srm.tracer, saveSubscription)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return srm.repo.Save(ctx, sub)
}

func (srm subscriptionRepositoryMiddleware) RetrieveByGroup(ctx context.Context, pm mqtt.PageMetadata, groupID string) (mqtt.Page, error) {
	span := dbutil.CreateSpan(ctx, srm.tracer, retrieveSubscriptionByGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return srm.repo.RetrieveByGroup(ctx, pm, groupID)
}

func (srm subscriptionRepositoryMiddleware) Remove(ctx context.Context, sub mqtt.Subscription) error {
	span := dbutil.CreateSpan(ctx, srm.tracer, removeSubscription)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return srm.repo.Remove(ctx, sub)
}

func (srm subscriptionRepositoryMiddleware) RemoveByThing(ctx context.Context, thingID string) error {
	span := dbutil.CreateSpan(ctx, srm.tracer, removeSubscriptionByThing)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return srm.repo.RemoveByThing(ctx, thingID)
}

func (srm subscriptionRepositoryMiddleware) RemoveByGroup(ctx context.Context, groupID string) error {
	span := dbutil.CreateSpan(ctx, srm.tracer, removeSubscriptionByGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return srm.repo.RemoveByGroup(ctx, groupID)
}
