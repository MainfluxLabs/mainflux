package webhooks

import "context"

type Webhook struct {
	ThingID string
	Name    string
	Url     string
}

type WebhookRepository interface {
	// Save persists multiple webhooks. Webhooks are saved using a transaction.
	// If one webhook fails then none will be saved.
	// Successful operation is indicated by non-nil error response.
	Save(ctx context.Context, whs ...Webhook) ([]Webhook, error)

	// RetrieveByThingID retrieves webhooks related to
	// a certain thing identified by a given ID.
	RetrieveByThingID(ctx context.Context, thingID string) ([]Webhook, error)
}
