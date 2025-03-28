package alarms

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

type Alarm struct {
	ID       string
	ThingID  string
	GroupID  string
	Subtopic string
	Protocol string
	Payload  map[string]interface{}
	Rule     map[string]interface{}
	Created  int64
}

type AlarmsPage struct {
	apiutil.PageMetadata
	Alarms []Alarm
}

// AlarmRepository specifies an alarm persistence API.
type AlarmRepository interface {
	// Save persists multiple alarms. Alarms are saved using a transaction.
	// If one alarm fails, none will be saved. A successful operation is indicated by a non-nil error response.
	Save(ctx context.Context, alarms ...Alarm) error

	// RetrieveByID retrieves an alarm by its unique identifier.
	RetrieveByID(ctx context.Context, id string) (Alarm, error)

	// RetrieveByThing retrieves alarms associated with a given thing ID.
	RetrieveByThing(ctx context.Context, thingID string, pm apiutil.PageMetadata) (AlarmsPage, error)

	// RetrieveByGroup retrieves alarms associated with a given group ID.
	RetrieveByGroup(ctx context.Context, groupID string, pm apiutil.PageMetadata) (AlarmsPage, error)

	// Remove removes alarms by their identifiers.
	Remove(ctx context.Context, ids ...string) error
}
