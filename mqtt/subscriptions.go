package mqtt

import "context"

// Subscription represents a user Subscription.
type Subscription struct {
	Subtopic  string
	ThingID   string
	GroupID   string
	CreatedAt float64
}

// Page represents page metadata with content.
type Page struct {
	Total         uint64
	Subscriptions []Subscription
}

// PageMetadata contains page metadata that helps navigation.
type PageMetadata struct {
	Offset uint64
	Total  uint64
	Limit  uint64
}

type Repository interface {
	// Save will save the subscription.
	Save(ctx context.Context, sub Subscription) error

	// RetrieveByGroup retrieves all subscriptions that belong to the specified group.
	RetrieveByGroup(ctx context.Context, pm PageMetadata, groupID string) (Page, error)

	// Remove will remove the subscription.
	Remove(ctx context.Context, sub Subscription) error

	// RemoveByThing removes all subscriptions associated with the specified thing ID.
	RemoveByThing(ctx context.Context, thingID string) error

	// RemoveByGroup removes all subscriptions associated with the specified group ID.
	RemoveByGroup(ctx context.Context, groupID string) error
}
