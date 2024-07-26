package tracing

import (
	"context"

	"github.com/MainfluxLabs/mainflux/webhooks"
	"github.com/opentracing/opentracing-go"
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
	span := createSpan(ctx, wrm.tracer, "save_webhooks")
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return wrm.repo.Save(ctx, whs...)
}

func (wrm webhookRepositoryMiddleware) RetrieveByGroupID(ctx context.Context, groupID string) ([]webhooks.Webhook, error) {
	span := createSpan(ctx, wrm.tracer, "retrieve_by_group_id")
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return wrm.repo.RetrieveByGroupID(ctx, groupID)
}

func (wrm webhookRepositoryMiddleware) RetrieveByID(ctx context.Context, id string) (webhooks.Webhook, error) {
	span := createSpan(ctx, wrm.tracer, "retrieve_webhook_by_id")
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return wrm.repo.RetrieveByID(ctx, id)
}

func (wrm webhookRepositoryMiddleware) Update(ctx context.Context, w webhooks.Webhook) error {
	span := createSpan(ctx, wrm.tracer, "update_webhook")
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return wrm.repo.Update(ctx, w)
}

func (wrm webhookRepositoryMiddleware) Remove(ctx context.Context, groupID string, ids ...string) error {
	span := createSpan(ctx, wrm.tracer, "remove_webhooks")
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return wrm.repo.Remove(ctx, groupID, ids...)
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
