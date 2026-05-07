package ui

import (
	"bytes"
	"os"
	"testing"

	"encoding/json"
	"github.com/wasilak/dotisan/pkg/engine"
	"github.com/wasilak/dotisan/pkg/output"
	"github.com/wasilak/dotisan/pkg/provider"
)

// Test that DisplayPlanResult renders provider warnings into stdout
func TestDisplayPlanResult_RendersWarnings(t *testing.T) {
	// Build a PlanResult with a single provider plan that contains a warning
	pr := &engine.PlanResult{
		ProviderPlans: map[string]provider.GroupPlan{
			"file": {
				Warnings: []provider.PlanWarning{
					{
						GroupID:    "ManagedFile/myfiles",
						ItemID:     "zshrc",
						Severity:   "warning",
						Message:    "Destination file already exists at ~/.zshrc",
						Suggestion: "dotisan state import ManagedFile myfiles ~/.zshrc",
					},
				},
			},
		},
		TotalAdditions: 0,
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run
	if err := DisplayPlanResult(pr, output.FormatPlain, false); err != nil {
		t.Fatalf("display failed: %v", err)
	}

	// Restore and read
	_ = w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	os.Stdout = old

	out := buf.String()
	if !bytes.Contains([]byte(out), []byte("Warnings")) {
		t.Fatalf("expected output to contain Warnings header, got:\n%s", out)
	}
	if !bytes.Contains([]byte(out), []byte("Destination file already exists")) {
		t.Fatalf("expected warning message present, got:\n%s", out)
	}
	if !bytes.Contains([]byte(out), []byte("dotisan state import ManagedFile myfiles ~/.zshrc")) {
		t.Fatalf("expected suggestion present, got:\n%s", out)
	}
}

// Test that DisplayPlanResult includes warnings in JSON output
func TestDisplayPlanResult_JSONIncludesWarnings(t *testing.T) {
	pr := &engine.PlanResult{
		ProviderPlans: map[string]provider.GroupPlan{
			"file": {
				Warnings: []provider.PlanWarning{
					{
						GroupID:    "ManagedFile/myfiles",
						ItemID:     "zshrc",
						Severity:   "warning",
						Message:    "Destination file already exists at ~/.zshrc",
						Suggestion: "dotisan state import ManagedFile myfiles ~/.zshrc",
					},
				},
			},
		},
		TotalAdditions: 0,
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	if err := DisplayPlanResult(pr, output.FormatJSON, false); err != nil {
		t.Fatalf("display failed: %v", err)
	}

	_ = w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	os.Stdout = old

	var out PlanOutput
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid json output: %v; raw: %s", err, buf.String())
	}

	// top-level warnings should be present
	if len(out.Warnings) == 0 {
		t.Fatalf("expected warnings array in json output, got none")
	}

	// check first warning fields
	first := out.Warnings[0]
	if first.Provider != "file" {
		t.Fatalf("expected provider 'file', got: %v", first.Provider)
	}
	if first.GroupID != "ManagedFile/myfiles" {
		t.Fatalf("expected group_id, got: %v", first.GroupID)
	}
	if first.ItemID != "zshrc" {
		t.Fatalf("expected item_id zshrc, got: %v", first.ItemID)
	}
	if first.Message == "" {
		t.Fatalf("expected message to be present")
	}
	if first.Suggestion == "" {
		t.Fatalf("expected suggestion to be present")
	}
}

// Test that tree output includes provider warnings
func TestDisplayPlanResult_TreeIncludesWarnings(t *testing.T) {
	pr := &engine.PlanResult{
		ProviderPlans: map[string]provider.GroupPlan{
			"file": {
				Warnings: []provider.PlanWarning{
					{
						GroupID:    "ManagedFile/myfiles",
						ItemID:     "zshrc",
						Severity:   "warning",
						Message:    "Destination file already exists at ~/.zshrc",
						Suggestion: "dotisan state import ManagedFile myfiles ~/.zshrc",
					},
				},
			},
		},
		TotalAdditions: 0,
	}

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	if err := DisplayPlanResult(pr, output.FormatTree, false); err != nil {
		t.Fatalf("display failed: %v", err)
	}

	_ = w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	os.Stdout = old

	out := buf.String()
	// The output may include ANSI styling. Check for unstyled substrings to be
	// tolerant to terminal colouring.
	if !bytes.Contains([]byte(out), []byte("ManagedFile/myfiles")) {
		t.Fatalf("expected tree output to contain group id, got:\n%s", out)
	}
}
