package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
	"github.com/MainfluxLabs/mainflux/webhooks"
	"github.com/opentracing/opentracing-go"
)

const (
	saveWebhooks            = "save_webhooks"
	retrieveWebhooksByGroup = "retrieve_webhooks_by_group"
	retrieveWebhooksByThing = "retrieve_webhooks_by_thing"
	retrieveWebhookByID     = "retrieve_webhook_by_id"
	updateWebhook           = "update_webhook"
	removeWebhooks          = "remove_webhooks"
)

var (
	_ webhooks.WebhookRepository = (*webhookRepositoryMiddleware)(nil)
)

type webhookRepositoryMiddleware struct {
	tracer opentracing.Tracer
	repo   webhooks.WebhookRepository
}

// WebhookRepositoryMiddleware tracks request and their latency, and adds spans
// to context.
func WebhookRepositoryMiddleware(tracer opentracing.Tracer, repo webhooks.WebhookRepository) webhooks.WebhookRepository {
	return webhookRepositoryMiddleware{
		tracer: tracer,
		repo:   repo,
	}
}

func (wrm webhookRepositoryMiddleware) Save(ctx context.Context, whs ...webhooks.Webhook) ([]webhooks.Webhook, error) {
	span := createSpan(ctx, wrm.tracer, saveWebhooks)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return wrm.repo.Save(ctx, whs...)
}

func (wrm webhookRepositoryMiddleware) RetrieveByGroup(ctx context.Context, groupID string, pm apiutil.PageMetadata) (webhooks.WebhooksPage, error) {
	span := createSpan(ctx, wrm.tracer, retrieveWebhooksByGroup)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return wrm.repo.RetrieveByGroup(ctx, groupID, pm)
}

func (wrm webhookRepositoryMiddleware) RetrieveByThing(ctx context.Context, thingID string, pm apiutil.PageMetadata) (webhooks.WebhooksPage, error) {
	span := createSpan(ctx, wrm.tracer, retrieveWebhooksByThing)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return wrm.repo.RetrieveByThing(ctx, thingID, pm)
}

func (wrm webhookRepositoryMiddleware) RetrieveByID(ctx context.Context, id string) (webhooks.Webhook, error) {
	span := createSpan(ctx, wrm.tracer, retrieveWebhookByID)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return wrm.repo.RetrieveByID(ctx, id)
}

func (wrm webhookRepositoryMiddleware) Update(ctx context.Context, w webhooks.Webhook) error {
	span := createSpan(ctx, wrm.tracer, updateWebhook)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return wrm.repo.Update(ctx, w)
}

func (wrm webhookRepositoryMiddleware) Remove(ctx context.Context, ids ...string) error {
	span := createSpan(ctx, wrm.tracer, removeWebhooks)
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return wrm.repo.Remove(ctx, ids...)
}

func createSpan(ctx context.Context, tracer opentracing.Tracer, opName string) opentracing.Span {
	if parentSpan := opentracing.SpanFromContext(ctx); parentSpan != nil {
		return tracer.StartSpan(
			opName,
			opentracing.ChildOf(parentSpan.Context()),
		)
	}
	return tracer.StartSpan(opName)
}
