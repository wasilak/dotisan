package audit

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/wasilak/nim/pkg/style"
)

// ReportText renders a human-readable report
func ReportText(r *AuditResult, min Severity) string {
	var sb strings.Builder
	sb.WriteString(style.Header.Render("Resource Manifest Audit") + "\n")
	sb.WriteString(fmt.Sprintf("  Total resources: %d\n", r.Summary.TotalResources))
	sb.WriteString(fmt.Sprintf("  Valid: %d, Errors: %d, Warnings: %d\n\n", r.Summary.Valid, r.Summary.Errors, r.Summary.Warnings))

	for _, it := range r.Issues {
		if severityLessThan(it.Severity, min) {
			continue
		}
		sev := string(it.Severity)
		icon := style.IconWarning
		if it.Severity == SeverityError {
			icon = style.IconError
		}
		sb.WriteString(fmt.Sprintf("%s %s: %s\n", icon, strings.ToUpper(sev), it.Message))
		if it.File != "" {
			sb.WriteString(fmt.Sprintf("    File: %s\n", it.File))
		}
		if it.Resource != "" {
			sb.WriteString(fmt.Sprintf("    Resource: %s\n", it.Resource))
		}
		if it.Suggestion != "" {
			sb.WriteString(fmt.Sprintf("    Suggestion: %s\n", it.Suggestion))
		}
		sb.WriteString("\n")
	}

	// Print simple dependency summary
	sb.WriteString("Dependency Graph Summary:\n")
	sb.WriteString(fmt.Sprintf("  Nodes: %d, Edges: %d\n", len(r.DependencyGraph.Nodes), len(r.DependencyGraph.Edges)))
	return sb.String()
}

func severityLessThan(s Severity, min Severity) bool {
	order := map[Severity]int{SeverityInfo: 0, SeverityWarning: 1, SeverityError: 2}
	return order[s] < order[min]
}

// ReportJSON returns a JSON payload for the result
func ReportJSON(r *AuditResult) ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
