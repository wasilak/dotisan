package diff

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// PlanFormatter provides helper functions for formatting plan output.
type PlanFormatter struct {
	styles Styles
}

// NewPlanFormatter creates a new PlanFormatter with default styles.
func NewPlanFormatter() *PlanFormatter {
	return &PlanFormatter{
		styles: DefaultStyles(),
	}
}

// SetStyles allows customizing the color scheme.
func (f *PlanFormatter) SetStyles(styles Styles) {
	f.styles = styles
}

// FormatAddition formats an addition message.
func (f *PlanFormatter) FormatAddition(resourceName string) string {
	prefix := lipgloss.NewStyle().
		Foreground(lipgloss.Color("42")).
		Bold(true).
		Render("+")

	return fmt.Sprintf("%s %s", prefix, resourceName)
}

// FormatDeletion formats a deletion message.
func (f *PlanFormatter) FormatDeletion(resourceName string) string {
	prefix := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true).
		Render("-")

	return fmt.Sprintf("%s %s", prefix, resourceName)
}

// FormatModification formats a modification message with optional diff.
func (f *PlanFormatter) FormatModification(resourceName string, diff string) string {
	prefix := lipgloss.NewStyle().
		Foreground(lipgloss.Color("226")).
		Bold(true).
		Render("~")

	if diff != "" {
		// Show diff preview
		lines := strings.Split(diff, "\n")
		if len(lines) > 3 {
			lines = lines[:3]
			lines = append(lines, "...")
		}
		return fmt.Sprintf("%s %s\n%s", prefix, resourceName, strings.Join(lines, "\n"))
	}

	return fmt.Sprintf("%s %s", prefix, resourceName)
}

// FormatInSync formats an in-sync message.
func (f *PlanFormatter) FormatInSync(resourceName string) string {
	prefix := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")). // Dim gray
		Render("=")

	return fmt.Sprintf("%s %s", prefix, resourceName)
}

// FormatDrift formats a drift detection message.
// The description can be multiline (for diffs), and will be properly indented.
func (f *PlanFormatter) FormatDrift(resourceName, description string) string {
	prefix := lipgloss.NewStyle().
		Foreground(lipgloss.Color("208")). // Orange
		Bold(true).
		Render("!")

	if description == "" {
		return fmt.Sprintf("%s %s (drift detected)", prefix, resourceName)
	}

	// Check if description is multiline (contains \n)
	if !strings.Contains(description, "\n") {
		// Single line - use simple format
		return fmt.Sprintf("%s %s (drift: %s)", prefix, resourceName, description)
	}

	// Multiline description (likely a diff) - format with proper indentation
	lines := strings.Split(description, "\n")
	var formattedLines []string
	
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		// Apply color based on diff prefix
		if strings.HasPrefix(line, "+") {
			// Addition - green
			colored := lipgloss.NewStyle().
				Foreground(lipgloss.Color("42")). // Green
				Render(line)
			formattedLines = append(formattedLines, "    "+colored)
		} else if strings.HasPrefix(line, "-") {
			// Deletion - red
			colored := lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")). // Red
				Render(line)
			formattedLines = append(formattedLines, "    "+colored)
		} else {
			// Context line
			formattedLines = append(formattedLines, "    "+line)
		}
	}

	return fmt.Sprintf("%s %s (drift detected):\n%s", prefix, resourceName, strings.Join(formattedLines, "\n"))
}

// FormatSummary formats a plan summary with counts.
func (f *PlanFormatter) FormatSummary(add, modify, remove, inSync int) string {
	var parts []string

	if add > 0 {
		addStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
		parts = append(parts, addStyle.Render(fmt.Sprintf("+%d to add", add)))
	}

	if modify > 0 {
		modifyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
		parts = append(parts, modifyStyle.Render(fmt.Sprintf("~%d to change", modify)))
	}

	if remove > 0 {
		removeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		parts = append(parts, removeStyle.Render(fmt.Sprintf("-%d to remove", remove)))
	}

	if inSync > 0 {
		syncStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		parts = append(parts, syncStyle.Render(fmt.Sprintf("=%d unchanged", inSync)))
	}

	if len(parts) == 0 {
		return "No changes"
	}

	return "Plan: " + strings.Join(parts, ", ")
}

// FormatResourceHeader formats a resource section header.
func (f *PlanFormatter) FormatResourceHeader(kind, name string) string {
	kindStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("255")).
		Render(kind)

	nameStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("250")).
		Render(name)

	return fmt.Sprintf("%s/%s", kindStyle, nameStyle)
}
