package mqtt

import "context"

// Subscription represents a user Subscription.
type Subscription struct {
	Subtopic  string
	ThingID   string
	GroupID   string
	ClientID  string
	Status    string
	CreatedAt float64
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
	// RetrieveByGroupID retrieves all subscriptions that belong to the specified group.
	RetrieveByGroupID(ctx context.Context, pm PageMetadata, groupID string) (Page, error)
	// Save will save the subscription.
	Save(ctx context.Context, sub Subscription) error
	// Remove will remove the subscription.
	Remove(ctx context.Context, sub Subscription) error
	// Update will update the subscription status.
	UpdateStatus(ctx context.Context, sub Subscription) error
	// HasClientID will update the subscription status.
	HasClientID(ctx context.Context, clientID string) error
}
