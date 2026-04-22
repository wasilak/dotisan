package diff

import (
	"strings"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/wasilak/dotisan/pkg/style"
)

// Styles holds lipgloss styles for diff output.
type Styles struct {
	// Addition style for new lines
	Addition lipgloss.Style

	// Deletion style for removed lines
	Deletion lipgloss.Style

	// Modification style for changed lines
	Modification lipgloss.Style

	// Unchanged style for context lines
	Unchanged lipgloss.Style

	// Header style for diff headers
	Header lipgloss.Style
}

// DefaultStyles returns the default color scheme for diffs.
// Uses color constants from pkg/style for consistency.
func DefaultStyles() Styles {
	return Styles{
		Addition: lipgloss.NewStyle().
			Foreground(lipgloss.Color(style.Green)).
			Background(lipgloss.Color("22")),

		Deletion: lipgloss.NewStyle().
			Foreground(lipgloss.Color(style.Red)).
			Background(lipgloss.Color("52")),

		Modification: lipgloss.NewStyle().
			Foreground(lipgloss.Color(style.Yellow)).
			Background(lipgloss.Color("58")),

		Unchanged: lipgloss.NewStyle().
			Foreground(lipgloss.Color(style.Gray)),

		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("255")),
	}
}

// StyledDiffEngine extends Engine with styling capabilities.
type StyledDiffEngine struct {
	*Engine
	styles Styles
}

// NewStyledEngine creates a new StyledDiffEngine with default styles.
func NewStyledEngine() *StyledDiffEngine {
	return &StyledDiffEngine{
		Engine: NewEngine(),
		styles: DefaultStyles(),
	}
}

// SetStyles allows customizing the color scheme.
func (e *StyledDiffEngine) SetStyles(styles Styles) {
	e.styles = styles
}

// FormatLineChanges formats line changes with colors.
func (e *StyledDiffEngine) FormatLineChanges(changes []LineChange) string {
	var result strings.Builder

	for _, change := range changes {
		prefix := change.Type.String()
		content := change.Content
		fullLine := prefix + " " + content

		switch change.Type {
		case LineAdded:
			result.WriteString(e.styles.Addition.Render(fullLine))
		case LineDeleted:
			result.WriteString(e.styles.Deletion.Render(fullLine))
		case LineModified:
			result.WriteString(e.styles.Modification.Render(fullLine))
		default:
			result.WriteString(e.styles.Unchanged.Render(fullLine))
		}
		result.WriteString("\n")
	}

	return result.String()
}

// HighlightIntraLineChanges shows character-level differences within modified lines.
func (e *StyledDiffEngine) HighlightIntraLineChanges(oldLine, newLine string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(oldLine, newLine, false)

	var result strings.Builder
	for _, diff := range diffs {
		text := diff.Text
		switch diff.Type {
		case diffmatchpatch.DiffDelete:
			result.WriteString(e.styles.Deletion.Render(text))
		case diffmatchpatch.DiffInsert:
			result.WriteString(e.styles.Addition.Render(text))
		case diffmatchpatch.DiffEqual:
			result.WriteString(e.styles.Unchanged.Render(text))
		}
	}

	return result.String()
}

// FormatUnifiedDiff applies styling to a unified diff.
func (e *StyledDiffEngine) FormatUnifiedDiff(diff string) string {
	lines := strings.Split(diff, "\n")
	var result strings.Builder

	for _, line := range lines {
		if len(line) == 0 {
			result.WriteString("\n")
			continue
		}

		// Check prefix
		prefix := line[0]
		content := line[1:]

		switch prefix {
		case '+':
			if strings.HasPrefix(content, "+++") {
				// File header
				result.WriteString(e.styles.Header.Render(line))
			} else {
				result.WriteString(e.styles.Addition.Render(line))
			}
		case '-':
			if strings.HasPrefix(content, "---") {
				// File header
				result.WriteString(e.styles.Header.Render(line))
			} else {
				result.WriteString(e.styles.Deletion.Render(line))
			}
		case '@':
			// Chunk header
			result.WriteString(e.styles.Modification.Render(line))
		default:
			result.WriteString(e.styles.Unchanged.Render(line))
		}
		result.WriteString("\n")
	}

	return result.String()
}
