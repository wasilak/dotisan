package diff

import (
	"fmt"
	"strings"

	"github.com/wasilak/nim/pkg/style"
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

// Icons for different change types
const (
	IconAddition     = "✚"
	IconDeletion     = "✖"
	IconModification = "✎"
	IconDrift        = "⚠"
	IconInSync       = "✓"
)

// PlanFormatter uses the shared style package for coloring so the
// plan output matches other CLI outputs. Colors live in pkg/style.

// FormatAddition formats an addition message.
func (f *PlanFormatter) FormatAddition(resourceName string) string {
	// Use shared styled icon and row success color
	icon := style.StyledIconAdd
	name := style.RowSuccess.Render(resourceName)
	return fmt.Sprintf("%s %s", icon, name)
}

// FormatDeletion formats a deletion message.
func (f *PlanFormatter) FormatDeletion(resourceName string) string {
	icon := style.IconError
	name := style.RowError.Render(resourceName)
	return fmt.Sprintf("%s %s", icon, name)
}

// FormatModification formats a modification message with optional diff.
func (f *PlanFormatter) FormatModification(resourceName string, diff string) string {
	icon := style.Info.Render(IconModification)
	name := style.RowWarning.Render(resourceName)

	if diff != "" {
		// Format multiline diff properly
		formattedDiff := f.formatMultilineDiff(diff, "  ")
		return fmt.Sprintf("%s %s\n%s", icon, name, formattedDiff)
	}

	return fmt.Sprintf("%s %s", icon, name)
}

// FormatInSync formats an in-sync message.
func (f *PlanFormatter) FormatInSync(resourceName string) string {
	icon := style.DimStyle.Render(IconInSync)
	name := style.DimStyle.Render(resourceName)
	return fmt.Sprintf("%s %s", icon, name)
}

// FormatDrift formats a drift detection message.
// The description can be multiline (for diffs), and will be properly indented.
func (f *PlanFormatter) FormatDrift(resourceName, description string) string {
	icon := style.Warning.Render(IconDrift)
	name := style.RowWarning.Render(resourceName)

	if description == "" {
		return fmt.Sprintf("%s %s (drift detected)", icon, name)
	}

	// Check if description is multiline (contains \n)
	if !strings.Contains(description, "\n") {
		// Single line - use simple format
		return fmt.Sprintf("%s %s (drift: %s)", icon, name, description)
	}

	// Multiline description (likely a diff) - format with proper indentation
	formattedDiff := f.formatMultilineDiff(description, "  ")
	return fmt.Sprintf("%s %s (drift detected):\n%s", icon, name, formattedDiff)
}

// formatMultilineDiff formats a multiline diff with proper indentation and colors.
func (f *PlanFormatter) formatMultilineDiff(diff, indent string) string {
	lines := strings.Split(diff, "\n")
	var formattedLines []string

	for _, line := range lines {
		if line == "" {
			continue
		}

		// Apply color based on diff prefix (pastel colors)
		if strings.HasPrefix(line, "+") {
			// Addition - use shared row success color
			colored := style.RowSuccess.Render(line)
			formattedLines = append(formattedLines, indent+colored)
		} else if strings.HasPrefix(line, "-") {
			// Deletion - use shared row error color
			colored := style.RowError.Render(line)
			formattedLines = append(formattedLines, indent+colored)
		} else {
			// Context line - dim
			colored := style.DimStyle.Render(line)
			formattedLines = append(formattedLines, indent+colored)
		}
	}

	return strings.Join(formattedLines, "\n")
}

// FormatSummary formats a plan summary with counts.
func (f *PlanFormatter) FormatSummary(add, modify, remove, inSync int) string {
	var parts []string

	if add > 0 {
		parts = append(parts, style.Success.Render(fmt.Sprintf("+%d to add", add)))
	}

	if modify > 0 {
		parts = append(parts, style.Info.Render(fmt.Sprintf("~%d to change", modify)))
	}

	if remove > 0 {
		parts = append(parts, style.Error.Render(fmt.Sprintf("-%d to destroy", remove)))
	}

	// Note: Terraform doesn't show unchanged/in-sync resources in the summary
	// Only resources with actions (add/change/destroy) are shown

	if len(parts) == 0 {
		return "No changes. Your infrastructure matches the configuration."
	}

	return "Plan: " + strings.Join(parts, ", ")
}

// FormatWarningsSummary formats a short warnings summary string.
func (f *PlanFormatter) FormatWarningsSummary(warnCount int) string {
	if warnCount <= 0 {
		return ""
	}
	return style.Warning.Render(fmt.Sprintf("%s %d warnings", style.IconWarning, warnCount))
}

// FormatResourceHeader formats a resource section header.
func (f *PlanFormatter) FormatResourceHeader(kind, name string) string {
	kindStyle := style.Bold.Render(kind)
	nameStyle := style.DimStyle.Render(name)
	return fmt.Sprintf("%s/%s", kindStyle, nameStyle)
}

// Section header formatting (like Terraform)
func (f *PlanFormatter) FormatSectionHeader(title string) string {
	return style.Header.Render(title)
}

// Detailed formats with "will be..." text
func (f *PlanFormatter) FormatAdditionDetailed(resourceName string) string {
	icon := style.StyledIconAdd
	name := style.RowSuccess.Render(resourceName)
	action := style.DimStyle.Render("will be created")
	return fmt.Sprintf("  %s %s %s", icon, name, action)
}

func (f *PlanFormatter) FormatRemovalDetailed(resourceName string) string {
	icon := style.IconError
	name := style.RowError.Render(resourceName)
	action := style.DimStyle.Render("will be destroyed")
	return fmt.Sprintf("  %s %s %s", icon, name, action)
}

func (f *PlanFormatter) FormatModificationDetailed(resourceName string) string {
	icon := style.Info.Render(IconModification)
	name := style.RowWarning.Render(resourceName)
	action := style.DimStyle.Render("will be updated")
	return fmt.Sprintf("  %s %s %s", icon, name, action)
}

func (f *PlanFormatter) FormatDriftDetailed(resourceName string) string {
	icon := style.Warning.Render(IconDrift)
	name := style.RowWarning.Render(resourceName)
	action := style.DimStyle.Render("will be restored")
	return fmt.Sprintf("  %s %s %s", icon, name, action)
}

func (f *PlanFormatter) FormatInSyncDetailed(resourceName string) string {
	icon := style.DimStyle.Render(IconInSync)
	name := style.DimStyle.Render(resourceName)
	action := style.DimStyle.Render("no changes")
	return fmt.Sprintf("  %s %s %s", icon, name, action)
}

// FormatActionReason formats the reason text (gray, indented)
func (f *PlanFormatter) FormatActionReason(reason string) string {
	// If the reason includes a severity prefix like "WARNING: msg", highlight the
	// severity with the warning color and render the rest dimmed for readability.
	parts := strings.SplitN(reason, ": ", 2)
	if len(parts) == 2 {
		sev := style.Warning.Render(parts[0])
		msg := style.DimStyle.Render(parts[1])
		return fmt.Sprintf("%s: %s", sev, msg)
	}
	return style.DimStyle.Render(reason)
}

// FormatDiff formats a diff block
func (f *PlanFormatter) FormatDiff(diff string) string {
	return f.formatMultilineDiff(diff, "\t")
}

// FormatNoChanges formats the "no changes" message
func (f *PlanFormatter) FormatNoChanges() string {
	return style.DimStyle.Render("No changes. Your dotfiles are in sync!")
}
