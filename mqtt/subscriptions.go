package mqtt

import "context"

// Subscription represents a user Subscription.
type Subscription struct {
	ID       string
	OwnerID  string
	Subtopic string
	ThingID  string
	ChanID   string
}

// Page represents page metadata with content.
type Page struct {
	PageMetadata
	Subscriptions []Subscription
}

// PageMetadata contains page metadata that helps navigation.
type PageMetadata struct {
	Offset    uint64
	Total     uint64
	Limit     uint64
	Order     string
	Direction string
}

type Repository interface {
	// RetrieveAll retrieves all subscriptions.
	RetrieveAll(ctx context.Context, pm PageMetadata) (Page, error)
	// Save will save the subscription.
	Save(ctx context.Context, sub Subscription) (string, error)
	// Remove will remove the subscription.
	Remove(ctx context.Context, id string) error
}
