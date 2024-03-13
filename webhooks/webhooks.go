package webhooks

import "context"

type Webhook struct {
	ThingID string
	Name    string
	Format  string
	Url     string
}

type WebhookRepository interface {
	// Save persists webhook. A non-nil error is returned to indicate
	// operation failure.
	Save(ctx context.Context, w Webhook) (Webhook, error)
	RetrieveByThingID(ctx context.Context, thingID string) ([]Webhook, error)
}
