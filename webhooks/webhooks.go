package webhooks

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

type Webhook struct {
	ID       string
	ThingID  string
	GroupID  string
	Name     string
	Url      string
	Headers  map[string]string
	Metadata map[string]any
}

type WebhooksPage struct {
	Total    uint64
	Webhooks []Webhook
}

type WebhookRepository interface {
	// Save persists multiple webhooks. Webhooks are saved using a transaction.
	// If one webhook fails then none will be saved.
	// Successful operation is indicated by non-nil error response.
	Save(ctx context.Context, whs ...Webhook) ([]Webhook, error)

	// RetrieveByGroup retrieves webhooks related to a certain group identified by a given ID.
	RetrieveByGroup(ctx context.Context, groupID string, pm apiutil.PageMetadata) (WebhooksPage, error)

	// RetrieveByThing retrieves webhooks related to a certain thing identified by a given ID.
	RetrieveByThing(ctx context.Context, thingID string, pm apiutil.PageMetadata) (WebhooksPage, error)

	// RetrieveByID retrieves the webhook having the provided ID.
	RetrieveByID(ctx context.Context, id string) (Webhook, error)

	// Update performs an update to the existing webhook.
	// A non-nil error is returned to indicate operation failure.
	Update(ctx context.Context, w Webhook) error

	// Remove removes the webhooks identified with the provided IDs.
	Remove(ctx context.Context, ids ...string) error

	// RemoveByThing removes webhooks related to the given thing ID.
	RemoveByThing(ctx context.Context, thingID string) error

	// RemoveByGroup removes webhooks related to the given group ID.
	RemoveByGroup(ctx context.Context, groupID string) error
}
