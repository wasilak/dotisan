package audit

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

type ValidationIssue struct {
	Severity   Severity `json:"severity"`
	Category   string   `json:"category"`
	File       string   `json:"file,omitempty"`
	Resource   string   `json:"resource,omitempty"`
	Message    string   `json:"message"`
	Suggestion string   `json:"suggestion,omitempty"`
}

type ResourceNode struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	File   string `json:"file,omitempty"`
	Status string `json:"status"`
}

type DependencyEdge struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Exists   bool   `json:"exists"`
	IsCyclic bool   `json:"is_cyclic"`
}

type DependencyGraph struct {
	Nodes []ResourceNode   `json:"nodes"`
	Edges []DependencyEdge `json:"edges"`
}

type AuditSummary struct {
	TotalResources int `json:"total_resources"`
	Valid          int `json:"valid"`
	Errors         int `json:"errors"`
	Warnings       int `json:"warnings"`
}

type AuditResult struct {
	TotalFiles      int               `json:"total_files"`
	ValidFiles      int               `json:"valid_files"`
	Issues          []ValidationIssue `json:"issues"`
	DependencyGraph DependencyGraph   `json:"dependency_graph"`
	Summary         AuditSummary      `json:"summary"`
}
