package diff

import (
	"strings"
	"testing"
)

func TestNewEngine(t *testing.T) {
	e := NewEngine()
	if e == nil {
		t.Fatal("NewEngine() returned nil")
	}
	if e.context != 3 {
		t.Errorf("default context = %d, want 3", e.context)
	}
}

func TestEngine_GenerateUnifiedDiff(t *testing.T) {
	e := NewEngine()

	oldContent := "line1\nline2\nline3"
	newContent := "line1\nmodified\nline3"

	diff, err := e.GenerateUnifiedDiff("old.txt", "new.txt", oldContent, newContent)
	if err != nil {
		t.Fatalf("GenerateUnifiedDiff() failed: %v", err)
	}

	if diff == "" {
		t.Error("GenerateUnifiedDiff() returned empty diff")
	}

	// Should contain file headers
	if !strings.Contains(diff, "--- old.txt") {
		t.Error("diff should contain old file header")
	}
	if !strings.Contains(diff, "+++ new.txt") {
		t.Error("diff should contain new file header")
	}
}

func TestEngine_GenerateDiff(t *testing.T) {
	e := NewEngine()

	oldContent := "line1\nline2\nline3"
	newContent := "line1\nmodified\nline3"

	changes := e.GenerateDiff(oldContent, newContent)

	if len(changes) == 0 {
		t.Error("GenerateDiff() returned no changes")
	}

	// Find the modified line (content in unified diff has leading space)
	foundDeleted := false
	foundAdded := false
	for _, change := range changes {
		if change.Type == LineDeleted {
			foundDeleted = true
		}
		if change.Type == LineAdded {
			foundAdded = true
		}
	}
	if !foundDeleted {
		t.Error("should have detected deleted line")
	}
	if !foundAdded {
		t.Error("should have detected added line")
	}
}

func TestChangeType_String(t *testing.T) {
	tests := []struct {
		changeType ChangeType
		want       string
	}{
		{LineUnchanged, " "},
		{LineAdded, "+"},
		{LineDeleted, "-"},
		{LineModified, "~"},
		{ChangeType(99), "?"},
	}

	for _, tt := range tests {
		got := tt.changeType.String()
		if got != tt.want {
			t.Errorf("ChangeType(%d).String() = %q, want %q", tt.changeType, got, tt.want)
		}
	}
}

func TestDefaultStyles(t *testing.T) {
	styles := DefaultStyles()

	// Check that styles can render (this verifies they're valid lipgloss styles)
	_ = styles.Addition.Render("test")
	_ = styles.Deletion.Render("test")
	_ = styles.Modification.Render("test")
	_ = styles.Unchanged.Render("test")
	_ = styles.Header.Render("test")
	// If we get here without panic, styles are valid
}

func TestNewStyledEngine(t *testing.T) {
	e := NewStyledEngine()
	if e == nil {
		t.Fatal("NewStyledEngine() returned nil")
	}
	if e.Engine == nil {
		t.Error("Engine is nil")
	}
}

func TestStyledEngine_HighlightIntraLineChanges(t *testing.T) {
	e := NewStyledEngine()

	oldLine := "hello world"
	newLine := "hello beautiful world"

	result := e.HighlightIntraLineChanges(oldLine, newLine)

	// Result should contain styled text
	if result == "" {
		t.Error("HighlightIntraLineChanges() returned empty string")
	}

	// Should contain unchanged parts
	if !strings.Contains(result, "hello") {
		t.Error("result should contain 'hello'")
	}
	if !strings.Contains(result, "world") {
		t.Error("result should contain 'world'")
	}
}

func TestPlanFormatter(t *testing.T) {
	f := NewPlanFormatter()

	// Test FormatAddition
	addition := f.FormatAddition("test-resource")
	if addition == "" {
		t.Error("FormatAddition() returned empty")
	}
	if !strings.Contains(addition, "test-resource") {
		t.Error("addition should contain resource name")
	}

	// Test FormatDeletion
	deletion := f.FormatDeletion("test-resource")
	if deletion == "" {
		t.Error("FormatDeletion() returned empty")
	}

	// Test FormatModification
	modification := f.FormatModification("test-resource", "diff content")
	if modification == "" {
		t.Error("FormatModification() returned empty")
	}

	// Test FormatInSync
	inSync := f.FormatInSync("test-resource")
	if inSync == "" {
		t.Error("FormatInSync() returned empty")
	}

	// Test FormatDrift
	drift := f.FormatDrift("test-resource", "changed content")
	if drift == "" {
		t.Error("FormatDrift() returned empty")
	}
	if !strings.Contains(drift, "drift") {
		t.Error("drift should contain 'drift'")
	}
}

func TestPlanFormatter_FormatSummary(t *testing.T) {
	f := NewPlanFormatter()

	// Test with changes (Terraform-style: only shows add/change/destroy, not unchanged)
	summary := f.FormatSummary(2, 1, 1, 5)
	if summary == "" {
		t.Error("FormatSummary() returned empty")
	}
	if !strings.Contains(summary, "2 to add") {
		t.Error("summary should contain add count")
	}
	if !strings.Contains(summary, "1 to change") {
		t.Error("summary should contain change count")
	}
	if !strings.Contains(summary, "1 to destroy") {
		t.Error("summary should contain destroy count")
	}
	// Terraform doesn't show unchanged in summary - only resources with actions
	if strings.Contains(summary, "unchanged") {
		t.Error("summary should NOT contain unchanged count (Terraform-style)")
	}

	// Test with no changes
	noChange := f.FormatSummary(0, 0, 0, 0)
	if !strings.Contains(noChange, "No changes") {
		t.Errorf("FormatSummary(0,0,0,0) = %q, want message containing 'No changes'", noChange)
	}
}

func TestPlanFormatter_FormatResourceHeader(t *testing.T) {
	f := NewPlanFormatter()

	header := f.FormatResourceHeader("BrewPackages", "core-tools")
	if header == "" {
		t.Error("FormatResourceHeader() returned empty")
	}
	if !strings.Contains(header, "BrewPackages") {
		t.Error("header should contain kind")
	}
	if !strings.Contains(header, "core-tools") {
		t.Error("header should contain name")
	}
}
