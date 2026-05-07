package ui

import "github.com/wasilak/dotisan/pkg/provider"

// PlanOutput is the top-level structure marshalled for --output json.
type PlanOutput struct {
	Summary    SummaryStats            `json:"summary"`
	HasChanges bool                    `json:"has_changes"`
	Providers  map[string]ProviderPlan `json:"providers"`
	Warnings   []AggregatedWarning     `json:"warnings,omitempty"`
}

// SummaryStats holds the aggregate change counts across all providers.
type SummaryStats struct {
	Additions     int `json:"additions"`
	Modifications int `json:"modifications"`
	Removals      int `json:"removals"`
	Cleanup       int `json:"cleanup"`
	InSync        int `json:"in_sync"`
	Drifted       int `json:"drifted"`
}

// ProviderPlan holds the full per-provider plan data for machine consumption.
type ProviderPlan struct {
	Additions     []provider.GroupAddition     `json:"additions"`
	Modifications []provider.GroupModification `json:"modifications"`
	Removals      []provider.GroupRemoval      `json:"removals"`
	Cleanup       []provider.GroupCleanup      `json:"cleanup"`
	InSync        []provider.GroupState        `json:"in_sync"`
	Drifted       []provider.ItemDrift         `json:"drifted"`
	Warnings      []Warning                    `json:"warnings,omitempty"`
}

// Warning is a single provider-level advisory embedded inside a ProviderPlan.
type Warning struct {
	GroupID    string `json:"group_id"`
	ItemID     string `json:"item_id"`
	Severity   string `json:"severity"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion"`
}

// AggregatedWarning is a Warning annotated with its originating provider,
// collected into the top-level warnings array for easy machine consumption.
type AggregatedWarning struct {
	Provider   string `json:"provider"`
	GroupID    string `json:"group_id"`
	ItemID     string `json:"item_id"`
	Severity   string `json:"severity"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion"`
}
