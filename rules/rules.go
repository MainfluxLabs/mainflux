package rules

import (
	"context"

	"github.com/MainfluxLabs/mainflux/pkg/apiutil"
)

type Rule struct {
	ID          string
	ProfileID   string
	GroupID     string
	Name        string
	Description string
	Condition   Condition
	Actions     []Action
}

type Condition struct {
	Field     string   `json:"field"`
	Operator  string   `json:"operator"`
	Threshold *float64 `json:"threshold"`
}

type Action struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type RulesPage struct {
	apiutil.PageMetadata
	Rules []Rule
}

type RuleRepository interface {
	// Save persists multiple rules. Rules are saved using a transaction.
	// If one rule fails then none will be saved.
	// Successful operation is indicated by a non-nil error response.
	Save(ctx context.Context, rules ...Rule) ([]Rule, error)

	// RetrieveByID retrieves a rule having the provided identifier.
	RetrieveByID(ctx context.Context, id string) (Rule, error)

	// RetrieveByProfile retrieves rules assigned to a certain profile.
	RetrieveByProfile(ctx context.Context, profileID string, pm apiutil.PageMetadata) (RulesPage, error)

	// RetrieveByGroup retrieves rules assigned to a certain group.
	RetrieveByGroup(ctx context.Context, groupID string, pm apiutil.PageMetadata) (RulesPage, error)

	// Update performs an update to the existing rule. A non-nil error is
	// returned to indicate operation failure.
	Update(ctx context.Context, r Rule) error

	// Remove removes the rules having the provided identifiers.
	Remove(ctx context.Context, ids ...string) error
}
