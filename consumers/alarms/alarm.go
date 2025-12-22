package alarms

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

type Alarm struct {
	ID       string
	ThingID  string
	GroupID  string
	RuleID   string
	Subtopic string
	Protocol string
	Payload  map[string]any
	Created  int64
}

type AlarmsPage struct {
	Total  uint64
	Alarms []Alarm
}

// AlarmRepository specifies an alarm persistence API.
type AlarmRepository interface {
	// Save persists multiple alarms. Alarms are saved using a transaction.
	// If one alarm fails, none will be saved.
	// A successful operation is indicated by a non-nil error response.
	Save(ctx context.Context, alarms ...Alarm) error

	// RetrieveByID retrieves an alarm having the provided ID.
	RetrieveByID(ctx context.Context, id string) (Alarm, error)

	// RetrieveByThing retrieves alarms related to a certain thing,
	// identified by a given thing ID.
	RetrieveByThing(ctx context.Context, thingID string, pm apiutil.PageMetadata) (AlarmsPage, error)

	// RetrieveByGroup retrieves alarms related to a certain group,
	// identified by a given group ID.
	RetrieveByGroup(ctx context.Context, groupID string, pm apiutil.PageMetadata) (AlarmsPage, error)

	// RetrieveByGroups retrieves the subset of alarms related to groups identified by the given group IDs.
	RetrieveByGroups(ctx context.Context, groupIDs []string, pm apiutil.PageMetadata) (AlarmsPage, error)

	// Remove removes alarms having the provided IDs.
	Remove(ctx context.Context, ids ...string) error

	// RemoveByThing removes alarms related to a certain thing,
	// identified by a given thing ID.
	RemoveByThing(ctx context.Context, thingID string) error

	// RemoveByGroup removes alarms related to a certain group,
	// identified by a given group ID.
	RemoveByGroup(ctx context.Context, groupID string) error

	// BackupByThing backups alarms related to a certain thing,
	// identified by a given thing ID.
	BackupByThing(ctx context.Context, thingID string, pm apiutil.PageMetadata) (AlarmsPage, error)
}
