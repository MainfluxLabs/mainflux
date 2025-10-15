package rules

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

type Rule struct {
	ID          string
	ThingIDs    []string
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

	// RetrieveByID retrieves a rule having the provided identifier.
	RetrieveByID(ctx context.Context, id string) (Rule, error)

	// RetrieveByThing retrieves rules assigned to a certain thing.
	RetrieveByThing(ctx context.Context, thingID string, pm apiutil.PageMetadata) (RulesPage, error)

	// RetrieveByGroup retrieves rules assigned to a certain group.
	RetrieveByGroup(ctx context.Context, groupID string, pm apiutil.PageMetadata) (RulesPage, error)

	// Update performs an update to the existing rule. A non-nil error is
	// returned to indicate operation failure.
	Update(ctx context.Context, r Rule) error

	// Remove removes the rules having the provided identifiers.
	Remove(ctx context.Context, ids ...string) error

	// Assign assigns the specified rule to the given list of things.
	Assign(ctx context.Context, ruleID string, thingIDs []string) error

	// Unassign removes all thing associations for the given rule ID.
	Unassign(ctx context.Context, ruleID string) error
}
