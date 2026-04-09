package alarms

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/domain"
)

const (
	AlarmStatusActive  = "active"
	AlarmStatusNoted   = "noted"
	AlarmStatusCleared = "cleared"
)

type Alarm struct {
	ID       string
	ThingID  string
	GroupID  string
	RuleID   string
	ScriptID string
	Subtopic string
	Protocol string
	Payload  map[string]any
	Rule     *RuleInfo
	Level    int
	Status   string
	Created  int64
}

type AlarmsPage struct {
	Total  uint64
	Alarms []Alarm
}

// RuleInfo captures the evaluation logic of the rule that triggered an alarm.
type RuleInfo struct {
	Conditions []domain.Condition `json:"conditions"`
	Operator   string             `json:"operator,omitempty"`
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
	RetrieveByThing(ctx context.Context, thingID string, pm PageMetadata) (AlarmsPage, error)

	// RetrieveByGroup retrieves alarms related to a certain group,
	// identified by a given group ID.
	RetrieveByGroup(ctx context.Context, groupID string, pm PageMetadata) (AlarmsPage, error)

	// RetrieveByGroups retrieves the subset of alarms related to groups identified by the given group IDs.
	RetrieveByGroups(ctx context.Context, groupIDs []string, pm PageMetadata) (AlarmsPage, error)

	// Remove removes alarms having the provided IDs.
	Remove(ctx context.Context, ids ...string) error

	// RemoveByThing removes alarms related to a certain thing,
	// identified by a given thing ID.
	RemoveByThing(ctx context.Context, thingID string) error

	// RemoveByGroup removes alarms related to a certain group,
	// identified by a given group ID.
	RemoveByGroup(ctx context.Context, groupID string) error

	// UpdateStatus updates the status of an alarm identified by the provided ID.
	UpdateStatus(ctx context.Context, id, status string) error

	// ExportByThing retrieves alarms related to a certain thing,
	// identified by a given thing ID.
	ExportByThing(ctx context.Context, thingID string, pm PageMetadata) (AlarmsPage, error)
}
