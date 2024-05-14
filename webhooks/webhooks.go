package webhooks

import "context"

type Webhook struct {
	ID      string
	GroupID string
	Name    string
	Url     string
	Headers string
}

type WebhookRepository interface {
	// Save persists multiple webhooks. Webhooks are saved using a transaction.
	// If one webhook fails then none will be saved.
	// Successful operation is indicated by non-nil error response.
	Save(ctx context.Context, whs ...Webhook) ([]Webhook, error)

	// RetrieveByGroupID retrieves webhooks related to
	// a certain group identified by a given ID.
	RetrieveByGroupID(ctx context.Context, groupID string) ([]Webhook, error)
}
