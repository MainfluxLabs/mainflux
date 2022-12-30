package mqtt

import "context"

// Subscription represents a user Subscription.
type Subscription struct {
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
	Offset uint64
	Total  uint64
	Limit  uint64
}

type Repository interface {
	// RetrieveByOwnerID retrieves all subscriptions that belong to the specified owner.
	RetrieveByOwnerID(ctx context.Context, pm PageMetadata, ownerID string) (Page, error)
	// Save will save the subscription.
	Save(ctx context.Context, sub Subscription) error
	// Remove will remove the subscription.
	Remove(ctx context.Context, sub Subscription) error
}
