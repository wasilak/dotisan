// Package diff provides diff generation and formatting for dotisan.
//
// This package integrates multiple diff libraries:
//   - github.com/martinohmann/go-difflib: Line-level unified diffs
//   - github.com/sergi/go-diff: Character-level intra-line diffs
//   - github.com/charmbracelet/lipgloss: Colored terminal output
//
// The DiffEngine provides a unified interface for generating and formatting
// diffs suitable for the plan command output.
package diff

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/martinohmann/go-difflib/difflib"
)

// Engine provides diff generation capabilities.
type Engine struct {
	// unified context lines for diff output
	context int
}

// NewEngine creates a new DiffEngine with default settings.
func NewEngine() *Engine {
	return &Engine{
		context: 3, // Default 3 lines of context
	}
}

// SetContext sets the number of context lines for unified diffs.
func (e *Engine) SetContext(lines int) {
	e.context = lines
}

// GenerateUnifiedDiff creates a unified diff between two strings.
// Returns the diff in standard unified diff format.
func (e *Engine) GenerateUnifiedDiff(oldName, newName, oldContent, newContent string) (string, error) {
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	diff := difflib.UnifiedDiff{
		A:        oldLines,
		B:        newLines,
		FromFile: oldName,
		ToFile:   newName,
		Context:  e.context,
	}

	var buf bytes.Buffer
	if err := difflib.WriteUnifiedDiff(&buf, diff); err != nil {
		return "", fmt.Errorf("failed to generate diff: %w", err)
	}

	return buf.String(), nil
}

// GenerateDiff creates a simple line-by-line diff from unified diff output.
// Returns a list of changes with type (addition, deletion, unchanged).
func (e *Engine) GenerateDiff(oldContent, newContent string) []LineChange {
	// Generate unified diff first
	unifiedDiff, err := e.GenerateUnifiedDiff("old", "new", oldContent, newContent)
	if err != nil {
		return nil
	}

	// Parse the unified diff
	lines := strings.Split(unifiedDiff, "\n")
	var changes []LineChange

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		prefix := line[0]
		content := line[1:]

		switch prefix {
		case '+':
			// Skip "+++ filename" header lines but keep actual additions
			// Check if content starts with "++" (the remaining part of "+++")
			if !strings.HasPrefix(line, "+++") {
				changes = append(changes, LineChange{Type: LineAdded, Content: content})
			}
		case '-':
			// Skip "--- filename" header lines but keep actual deletions
			// Check if content starts with "--" (the remaining part of "---")
			if !strings.HasPrefix(line, "---") {
				changes = append(changes, LineChange{Type: LineDeleted, Content: content})
			}
		case ' ':
			changes = append(changes, LineChange{Type: LineUnchanged, Content: content})
		}
	}

	return changes
}

// LineChange represents a single line change in a diff.
type LineChange struct {
	Type    ChangeType
	Content string
}

// ChangeType indicates the type of line change.
type ChangeType int

const (
	LineUnchanged ChangeType = iota
	LineAdded
	LineDeleted
	LineModified
)

// String returns a string representation of the change type.
func (t ChangeType) String() string {
	switch t {
	case LineUnchanged:
		return " "
	case LineAdded:
		return "+"
	case LineDeleted:
		return "-"
	case LineModified:
		return "~"
	default:
		return "?"
	}
}
