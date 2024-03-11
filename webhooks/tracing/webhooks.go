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

func (wrm webhookRepositoryMiddleware) Save(ctx context.Context, w webhooks.Webhook) (webhooks.Webhook, error) {
	span := createSpan(ctx, wrm.tracer, "save_webhook")
	defer span.Finish()
	ctx = opentracing.ContextWithSpan(ctx, span)

	return wrm.repo.Save(ctx, w)
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
