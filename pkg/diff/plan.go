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

// Icons for different change types
const (
	IconAddition     = "✚"
	IconDeletion     = "✖"
	IconModification = "✎"
	IconDrift        = "⚠"
	IconInSync       = "✓"
)

// Pastel color palette (softer, easier on the eyes)
const (
	ColorPastelGreen  = "114" // Soft mint green
	ColorPastelRed    = "174" // Soft salmon/coral
	ColorPastelYellow = "222" // Soft cream yellow
	ColorPastelOrange = "216" // Soft peach
	ColorGray         = "240" // Neutral gray
)

// FormatAddition formats an addition message.
func (f *PlanFormatter) FormatAddition(resourceName string) string {
	icon := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorPastelGreen)).
		Render(IconAddition)

	nameStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorPastelGreen)).
		Render(resourceName)

	return fmt.Sprintf("%s %s", icon, nameStyle)
}

// FormatDeletion formats a deletion message.
func (f *PlanFormatter) FormatDeletion(resourceName string) string {
	icon := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorPastelRed)).
		Render(IconDeletion)

	nameStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorPastelRed)).
		Render(resourceName)

	return fmt.Sprintf("%s %s", icon, nameStyle)
}

// FormatModification formats a modification message with optional diff.
func (f *PlanFormatter) FormatModification(resourceName string, diff string) string {
	icon := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorPastelYellow)).
		Render(IconModification)

	nameStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorPastelYellow)).
		Render(resourceName)

	if diff != "" {
		// Format multiline diff properly
		formattedDiff := f.formatMultilineDiff(diff, "  ")
		return fmt.Sprintf("%s %s\n%s", icon, nameStyle, formattedDiff)
	}

	return fmt.Sprintf("%s %s", icon, nameStyle)
}

// FormatInSync formats an in-sync message.
func (f *PlanFormatter) FormatInSync(resourceName string) string {
	icon := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorGray)).
		Render(IconInSync)

	nameStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorGray)).
		Render(resourceName)

	return fmt.Sprintf("%s %s", icon, nameStyle)
}

// FormatDrift formats a drift detection message.
// The description can be multiline (for diffs), and will be properly indented.
func (f *PlanFormatter) FormatDrift(resourceName, description string) string {
	icon := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorPastelOrange)).
		Render(IconDrift)

	nameStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorPastelOrange)).
		Render(resourceName)

	if description == "" {
		return fmt.Sprintf("%s %s (drift detected)", icon, nameStyle)
	}

	// Check if description is multiline (contains \n)
	if !strings.Contains(description, "\n") {
		// Single line - use simple format
		return fmt.Sprintf("%s %s (drift: %s)", icon, nameStyle, description)
	}

	// Multiline description (likely a diff) - format with proper indentation
	formattedDiff := f.formatMultilineDiff(description, "  ")
	return fmt.Sprintf("%s %s (drift detected):\n%s", icon, nameStyle, formattedDiff)
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
			// Addition - soft mint green
			colored := lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorPastelGreen)).
				Render(line)
			formattedLines = append(formattedLines, indent+colored)
		} else if strings.HasPrefix(line, "-") {
			// Deletion - soft salmon
			colored := lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorPastelRed)).
				Render(line)
			formattedLines = append(formattedLines, indent+colored)
		} else {
			// Context line - soft gray
			colored := lipgloss.NewStyle().
				Foreground(lipgloss.Color("250")).
				Render(line)
			formattedLines = append(formattedLines, indent+colored)
		}
	}

	return strings.Join(formattedLines, "\n")
}

// FormatSummary formats a plan summary with counts.
func (f *PlanFormatter) FormatSummary(add, modify, remove, inSync int) string {
	var parts []string

	if add > 0 {
		addStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorPastelGreen))
		parts = append(parts, addStyle.Render(fmt.Sprintf("+%d to add", add)))
	}

	if modify > 0 {
		modifyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorPastelYellow))
		parts = append(parts, modifyStyle.Render(fmt.Sprintf("~%d to change", modify)))
	}

	if remove > 0 {
		removeStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorPastelRed))
		parts = append(parts, removeStyle.Render(fmt.Sprintf("-%d to destroy", remove)))
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
	warnStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorPastelOrange))
	return warnStyle.Render(fmt.Sprintf("⚠ %d warnings", warnCount))
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

// Section header formatting (like Terraform)
func (f *PlanFormatter) FormatSectionHeader(title string) string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("250")).
		Render(title)
}

// Detailed formats with "will be..." text
func (f *PlanFormatter) FormatAdditionDetailed(resourceName string) string {
	icon := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorPastelGreen)).
		Render(IconAddition)

	nameStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorPastelGreen)).
		Render(resourceName)

	action := lipgloss.NewStyle().
		Foreground(lipgloss.Color("250")).
		Render("will be created")

	return fmt.Sprintf("  %s %s %s", icon, nameStyle, action)
}

func (f *PlanFormatter) FormatRemovalDetailed(resourceName string) string {
	icon := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorPastelRed)).
		Render(IconDeletion)

	nameStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorPastelRed)).
		Render(resourceName)

	action := lipgloss.NewStyle().
		Foreground(lipgloss.Color("250")).
		Render("will be destroyed")

	return fmt.Sprintf("  %s %s %s", icon, nameStyle, action)
}

func (f *PlanFormatter) FormatModificationDetailed(resourceName string) string {
	icon := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorPastelYellow)).
		Render(IconModification)

	nameStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorPastelYellow)).
		Render(resourceName)

	action := lipgloss.NewStyle().
		Foreground(lipgloss.Color("250")).
		Render("will be updated")

	return fmt.Sprintf("  %s %s %s", icon, nameStyle, action)
}

func (f *PlanFormatter) FormatDriftDetailed(resourceName string) string {
	icon := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorPastelOrange)).
		Render(IconDrift)

	nameStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorPastelOrange)).
		Render(resourceName)

	action := lipgloss.NewStyle().
		Foreground(lipgloss.Color("250")).
		Render("will be restored")

	return fmt.Sprintf("  %s %s %s", icon, nameStyle, action)
}

func (f *PlanFormatter) FormatInSyncDetailed(resourceName string) string {
	icon := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorGray)).
		Render(IconInSync)

	nameStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorGray)).
		Render(resourceName)

	action := lipgloss.NewStyle().
		Foreground(lipgloss.Color("250")).
		Render("no changes")

	return fmt.Sprintf("  %s %s %s", icon, nameStyle, action)
}

// FormatActionReason formats the reason text (gray, indented)
func (f *PlanFormatter) FormatActionReason(reason string) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(reason)
}

// FormatDiff formats a diff block
func (f *PlanFormatter) FormatDiff(diff string) string {
	return f.formatMultilineDiff(diff, "      ")
}

// FormatNoChanges formats the "no changes" message
func (f *PlanFormatter) FormatNoChanges() string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("250")).
		Render("No changes. Your dotfiles are in sync!")
}
