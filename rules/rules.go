package rules

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

type Rule struct {
	ID          string
	GroupID     string
	Name        string
	Description string
	Conditions  []Condition
	Operator    string
	Actions     []Action
}

type Condition struct {
	Field      string   `json:"field"`
	Comparator string   `json:"comparator"`
	Threshold  *float64 `json:"threshold"`
}

type Action struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type RulesPage struct {
	Total uint64
	Rules []Rule
}

type RuleRepository interface {
	// Save persists multiple rules. Rules are saved using a transaction.
	// If one rule fails then none will be saved.
	// Successful operation is indicated by a non-nil error response.
	Save(ctx context.Context, rules ...Rule) ([]Rule, error)

	// RetrieveByID retrieves a rule having the provided ID.
	RetrieveByID(ctx context.Context, id string) (Rule, error)

	// RetrieveByThing retrieves rules assigned to a certain thing,
	// identified by a given thing ID.
	RetrieveByThing(ctx context.Context, thingID string, pm apiutil.PageMetadata) (RulesPage, error)

	// RetrieveByGroup retrieves rules related to a certain group,
	// identified by a given group ID.
	RetrieveByGroup(ctx context.Context, groupID string, pm apiutil.PageMetadata) (RulesPage, error)

	// RetrieveThingIDsByRule retrieves all thing IDs that have the given rule assigned.
	RetrieveThingIDsByRule(ctx context.Context, ruleID string) ([]string, error)

	// Update performs an update to the existing rule.
	// A non-nil error is returned to indicate operation failure.
	Update(ctx context.Context, r Rule) error

	// Remove removes rules having the provided IDs.
	Remove(ctx context.Context, ids ...string) error

	// RemoveByGroup removes rules related to a certain group,
	// identified by a given group ID.
	RemoveByGroup(ctx context.Context, groupID string) error

	// Assign assigns rules to the specified thing.
	Assign(ctx context.Context, thingID string, ruleIDs ...string) error

	// Unassign removes specific rule assignments from a given thing.
	Unassign(ctx context.Context, thingID string, ruleIDs ...string) error

	// UnassignByThing removes all rule assignments for a certain thing,
	// identified by a given thing ID.
	UnassignByThing(ctx context.Context, thingID string) error
}
