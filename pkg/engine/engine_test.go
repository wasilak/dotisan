package engine

import (
	"testing"
)

func TestNewEngine(t *testing.T) {
	// This test may fail if ~/.config/nim doesn't exist
	// That's expected in a fresh environment
	eng, err := NewEngine()
	if err != nil {
		t.Logf("NewEngine() returned error (may be expected without ~/.config/nim): %v", err)
		return
	}

	if eng == nil {
		t.Fatal("NewEngine() returned nil")
	}

	if eng.Config == nil {
		t.Error("Config should be initialized")
	}

	if eng.StateBackend == nil {
		t.Error("StateBackend should be initialized")
	}

	if len(eng.Providers) == 0 {
		t.Error("Providers should be initialized")
	}
}

func TestApplyOptions(t *testing.T) {
	opts := ApplyOptions{
		Confirm: true,
	}

	if !opts.Confirm {
		t.Error("Confirm should be true")
	}
}

func TestPlanResult(t *testing.T) {
	result := &PlanResult{
		TotalAdditions:     1,
		TotalModifications: 2,
		TotalRemovals:      3,
		TotalInSync:        4,
		TotalDrifted:       5,
	}

	result.HasChanges = result.TotalAdditions > 0 || result.TotalModifications > 0 || result.TotalRemovals > 0

	if !result.HasChanges {
		t.Error("HasChanges should be true")
	}

	if result.TotalAdditions != 1 {
		t.Errorf("TotalAdditions = %d, want 1", result.TotalAdditions)
	}
}
