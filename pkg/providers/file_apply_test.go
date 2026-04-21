package providers

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/resource"
)

func TestFileProvider_Apply_Addition(t *testing.T) {
	// Setup temp directories
	dotfilesDir := t.TempDir()
	destDir := t.TempDir()

	// Create resources subdirectory (files are resolved relative to resources/)
	resourcesDir := filepath.Join(dotfilesDir, "resources")
	os.MkdirAll(resourcesDir, 0755)

	// Create a source file in resources/
	sourceFile := filepath.Join(resourcesDir, "test.txt")
	content := "test content"
	if err := os.WriteFile(sourceFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Create provider
	p := NewFileProvider(nil, nil, dotfilesDir)

	// Create plan with addition
	mf := &resource.ManagedFile{
		BaseResource: resource.BaseResource{
			APIVersion: "github.com/wasilak/dotisan/v1",
			Kind:       "ManagedFile",
			Metadata:   resource.Metadata{Name: "test", Namespace: "default"},
		},
		Spec: resource.ManagedFileSpec{
			SourceFile:  "test.txt",
			Destination: filepath.Join(destDir, "dest.txt"),
			Template:    false,
			Mode:        "0644",
		},
	}

	plan := provider.Plan{
		Additions: []resource.Resource{mf},
	}

	// Apply
	ctx := context.Background()
	if err := p.Apply(ctx, plan); err != nil {
		t.Fatalf("Apply() failed: %v", err)
	}

	// Verify file was created
	destFile := filepath.Join(destDir, "dest.txt")
	if _, err := os.Stat(destFile); os.IsNotExist(err) {
		t.Error("destination file was not created")
	}

	// Verify content
	actualContent, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("Failed to read dest file: %v", err)
	}
	if string(actualContent) != content {
		t.Errorf("content = %q, want %q", string(actualContent), content)
	}
}

func TestFileProvider_Apply_Modification(t *testing.T) {
	// Setup temp directories
	dotfilesDir := t.TempDir()
	destDir := t.TempDir()

	// Create resources subdirectory (files are resolved relative to resources/)
	resourcesDir := filepath.Join(dotfilesDir, "resources")
	os.MkdirAll(resourcesDir, 0755)

	// Create a source file with new content in resources/
	sourceFile := filepath.Join(resourcesDir, "test.txt")
	newContent := "new content"
	if err := os.WriteFile(sourceFile, []byte(newContent), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Create destination file with old content
	destFile := filepath.Join(destDir, "dest.txt")
	oldContent := "old content"
	if err := os.WriteFile(destFile, []byte(oldContent), 0644); err != nil {
		t.Fatalf("Failed to create dest file: %v", err)
	}

	// Create provider
	p := NewFileProvider(nil, nil, dotfilesDir)

	// Create plan with modification
	mf := &resource.ManagedFile{
		BaseResource: resource.BaseResource{
			APIVersion: "github.com/wasilak/dotisan/v1",
			Kind:       "ManagedFile",
			Metadata:   resource.Metadata{Name: "test", Namespace: "default"},
		},
		Spec: resource.ManagedFileSpec{
			SourceFile:  "test.txt",
			Destination: destFile,
			Template:    false,
		},
	}

	plan := provider.Plan{
		Modifications: []provider.Modification{
			{
				Resource: mf,
				OldState: provider.ResourceState{DestHash: calculateChecksum(oldContent)},
				NewState: provider.ResourceState{DestHash: calculateChecksum(newContent)},
			},
		},
	}

	// Apply
	ctx := context.Background()
	if err := p.Apply(ctx, plan); err != nil {
		t.Fatalf("Apply() failed: %v", err)
	}

	// Verify content was updated
	actualContent, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatalf("Failed to read dest file: %v", err)
	}
	if string(actualContent) != newContent {
		t.Errorf("content = %q, want %q", string(actualContent), newContent)
	}
}

func TestFileProvider_Apply_Removal(t *testing.T) {
	// Setup temp directories
	destDir := t.TempDir()

	// Create destination file
	destFile := filepath.Join(destDir, "dest.txt")
	if err := os.WriteFile(destFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create dest file: %v", err)
	}

	// Create provider
	p := NewFileProvider(nil, nil, "")

	// Create plan with removal
	mf := &resource.ManagedFile{
		BaseResource: resource.BaseResource{
			APIVersion: "github.com/wasilak/dotisan/v1",
			Kind:       "ManagedFile",
			Metadata:   resource.Metadata{Name: "test", Namespace: "default"},
		},
		Spec: resource.ManagedFileSpec{
			SourceFile:  "test.txt",
			Destination: destFile,
			Template:    false,
		},
	}

	plan := provider.Plan{
		Removals: []resource.Resource{mf},
	}

	// Apply
	ctx := context.Background()
	if err := p.Apply(ctx, plan); err != nil {
		t.Fatalf("Apply() failed: %v", err)
	}

	// Verify file was removed
	if _, err := os.Stat(destFile); !os.IsNotExist(err) {
		t.Error("destination file should have been removed")
	}
}

func TestFileProvider_Apply_ContextCancellation(t *testing.T) {
	// Create provider
	p := NewFileProvider(nil, nil, "")

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Try to apply empty plan
	plan := provider.Plan{}
	err := p.Apply(ctx, plan)

	if err == nil {
		t.Error("Apply() should return error for cancelled context")
	}
}

func TestFileProvider_parseMode(t *testing.T) {
	p := NewFileProvider(nil, nil, "")

	tests := []struct {
		mode string
		want os.FileMode
	}{
		{"", 0644},        // Default
		{"0644", 0644},    // Standard file
		{"0755", 0755},    // Executable
		{"0600", 0600},    // Private
		{"0777", 0777},    // Full permissions
		{"invalid", 0644}, // Invalid falls back to default
		{"abc", 0644},     // Invalid falls back to default
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			got := p.parseMode(tt.mode)
			if got != tt.want {
				t.Errorf("parseMode(%q) = %o, want %o", tt.mode, got, tt.want)
			}
		})
	}
}

func TestFileProvider_Apply_CreatesParentDirectories(t *testing.T) {
	// Setup temp directories
	dotfilesDir := t.TempDir()
	destDir := t.TempDir()

	// Create resources subdirectory (files are resolved relative to resources/)
	resourcesDir := filepath.Join(dotfilesDir, "resources")
	os.MkdirAll(resourcesDir, 0755)

	// Create a source file in resources/
	sourceFile := filepath.Join(resourcesDir, "test.txt")
	if err := os.WriteFile(sourceFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Create provider
	p := NewFileProvider(nil, nil, dotfilesDir)

	// Destination with nested directories that don't exist
	destFile := filepath.Join(destDir, "a", "b", "c", "dest.txt")

	mf := &resource.ManagedFile{
		BaseResource: resource.BaseResource{
			APIVersion: "github.com/wasilak/dotisan/v1",
			Kind:       "ManagedFile",
			Metadata:   resource.Metadata{Name: "test", Namespace: "default"},
		},
		Spec: resource.ManagedFileSpec{
			SourceFile:  "test.txt",
			Destination: destFile,
			Template:    false,
		},
	}

	plan := provider.Plan{
		Additions: []resource.Resource{mf},
	}

	// Apply
	ctx := context.Background()
	if err := p.Apply(ctx, plan); err != nil {
		t.Fatalf("Apply() failed: %v", err)
	}

	// Verify file was created in nested directory
	if _, err := os.Stat(destFile); os.IsNotExist(err) {
		t.Error("destination file should have been created in nested directory")
	}
}
