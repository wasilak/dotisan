package providers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wasilak/dotisan/pkg/diff"
	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/resource"
)

// TestDiffEngineIntegration verifies that FileProvider correctly uses DiffEngine
func TestFileProvider_DiffEngineIntegration(t *testing.T) {
	// Setup temp directories
	dotfilesDir := t.TempDir()
	destDir := t.TempDir()

	// Create source file
	sourceFile := filepath.Join(dotfilesDir, "test.txt")
	if err := os.WriteFile(sourceFile, []byte("new content"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Create destination file with old content
	destFile := filepath.Join(destDir, "dest.txt")
	if err := os.WriteFile(destFile, []byte("old content"), 0644); err != nil {
		t.Fatalf("Failed to create dest file: %v", err)
	}

	// Create provider with DiffEngine
	diffEngine := diff.NewEngine()
	p := NewFileProvider(nil, diffEngine, dotfilesDir)

	// Create resource
	mf := &resource.ManagedFile{
		BaseResource: resource.BaseResource{
			APIVersion: "github.com/wasilak/dotisan/v1",
			Kind:       "ManagedFile",
			Metadata:   resource.Metadata{Name: "test", Namespace: "default"},
		},
		Spec: resource.ManagedFileSpec{
			Source:      "test.txt",
			Destination: destFile,
			Template:    false,
		},
	}

	// Reconcile
	desired := []resource.Resource{mf}
	state := []provider.ResourceState{
		{
			ID:       "file/default/test",
			Kind:     "ManagedFile",
			Name:     "test",
			DestHash: calculateChecksum("old content"),
		},
	}
	plan := p.Reconcile(desired, state)

	// Verify modification has diff
	if len(plan.Modifications) != 1 {
		t.Fatalf("len(Modifications) = %d, want 1", len(plan.Modifications))
	}

	mod := plan.Modifications[0]
	if mod.Diff == "" {
		t.Error("modification should have a diff generated")
	}
}

// TestStateTracking verifies that source_hash and dest_hash are correctly managed
func TestFileProvider_StateTracking(t *testing.T) {
	// Setup temp directories
	dotfilesDir := t.TempDir()
	destDir := t.TempDir()

	// Create source file
	sourceFile := filepath.Join(dotfilesDir, "test.txt")
	sourceContent := "source content"
	if err := os.WriteFile(sourceFile, []byte(sourceContent), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Create matching destination file
	destFile := filepath.Join(destDir, "dest.txt")
	if err := os.WriteFile(destFile, []byte(sourceContent), 0644); err != nil {
		t.Fatalf("Failed to create dest file: %v", err)
	}

	p := NewFileProvider(nil, nil, dotfilesDir)

	// Create resource
	mf := &resource.ManagedFile{
		BaseResource: resource.BaseResource{
			APIVersion: "github.com/wasilak/dotisan/v1",
			Kind:       "ManagedFile",
			Metadata:   resource.Metadata{Name: "test", Namespace: "default"},
		},
		Spec: resource.ManagedFileSpec{
			Source:      "test.txt",
			Destination: destFile,
			Template:    false,
		},
	}

	// Calculate expected hashes
	sourceHash := calculateChecksum(sourceContent)
	destHash := calculateChecksum(sourceContent)

	// Reconcile with matching state
	desired := []resource.Resource{mf}
	state := []provider.ResourceState{
		{
			ID:         "file/default/test",
			Kind:       "ManagedFile",
			Name:       "test",
			DestHash:   destHash,
			SourceHash: sourceHash,
		},
	}
	plan := p.Reconcile(desired, state)

	// Should be in sync when both hashes match
	if len(plan.InSync) != 1 {
		t.Errorf("len(InSync) = %d, want 1 (hashes should match)", len(plan.InSync))
	}
}

// TestDriftDetection verifies that drift is detected when file changes outside of dotisan
func TestFileProvider_DriftDetection(t *testing.T) {
	// Setup temp directories
	dotfilesDir := t.TempDir()
	destDir := t.TempDir()

	// Create source file
	sourceFile := filepath.Join(dotfilesDir, "test.txt")
	if err := os.WriteFile(sourceFile, []byte("original content"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Create destination file with different content (simulating external edit)
	destFile := filepath.Join(destDir, "dest.txt")
	if err := os.WriteFile(destFile, []byte("externally modified content"), 0644); err != nil {
		t.Fatalf("Failed to create dest file: %v", err)
	}

	p := NewFileProvider(nil, nil, dotfilesDir)

	// Create resource
	mf := &resource.ManagedFile{
		BaseResource: resource.BaseResource{
			APIVersion: "github.com/wasilak/dotisan/v1",
			Kind:       "ManagedFile",
			Metadata:   resource.Metadata{Name: "test", Namespace: "default"},
		},
		Spec: resource.ManagedFileSpec{
			Source:      "test.txt",
			Destination: destFile,
			Template:    false,
		},
	}

	// Reconcile with saved state matching original content
	desired := []resource.Resource{mf}
	state := []provider.ResourceState{
		{
			ID:       "file/default/test",
			Kind:     "ManagedFile",
			Name:     "test",
			DestHash: calculateChecksum("original content"), // Saved state
		},
	}
	plan := p.Reconcile(desired, state)

	// Should detect drift
	if len(plan.Drifted) != 1 {
		t.Errorf("len(Drifted) = %d, want 1 (should detect external modification)", len(plan.Drifted))
	}

	if len(plan.Drifted) > 0 && plan.Drifted[0].Description == "" {
		t.Error("drift should have a description")
	}
}
