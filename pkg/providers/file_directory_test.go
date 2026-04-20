package providers

import (
	"os"
	"path/filepath"
	"testing"

	"dotisan/pkg/provider"
	"dotisan/pkg/resource"
)

func TestFileProvider_Reconcile_ManagedDirectory_Addition(t *testing.T) {
	// Setup temp directories
	dotfilesDir := t.TempDir()
	destDir := t.TempDir()

	// Create a source directory with files
	sourceDir := filepath.Join(dotfilesDir, "skills")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "skill1.txt"), []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Create provider
	p := NewFileProvider(nil, nil, dotfilesDir)

	// Create desired resource
	md := &resource.ManagedDirectory{
		BaseResource: resource.BaseResource{
			APIVersion: "dotisan/v1",
			Kind:       "ManagedDirectory",
			Metadata:   resource.Metadata{Name: "skills", Namespace: "default"},
		},
		Spec: resource.ManagedDirectorySpec{
			Source:      "skills",
			Destination: filepath.Join(destDir, "skills"),
			Recursive:   false,
			Clean:       false,
		},
	}

	// Reconcile
	desired := []resource.Resource{md}
	state := []provider.ResourceState{}
	plan := p.Reconcile(desired, state)

	// Should have one addition
	if len(plan.Additions) != 1 {
		t.Errorf("len(Additions) = %d, want 1", len(plan.Additions))
	}
}

func TestFileProvider_Reconcile_ManagedDirectory_InSync(t *testing.T) {
	// Setup temp directories
	dotfilesDir := t.TempDir()
	destDir := t.TempDir()

	// Create a source directory
	sourceDir := filepath.Join(dotfilesDir, "skills")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "skill1.txt"), []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Create matching destination directory
	destSkillsDir := filepath.Join(destDir, "skills")
	if err := os.MkdirAll(destSkillsDir, 0755); err != nil {
		t.Fatalf("Failed to create dest dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(destSkillsDir, "skill1.txt"), []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create dest file: %v", err)
	}

	// Create provider
	p := NewFileProvider(nil, nil, dotfilesDir)

	// Create desired resource
	md := &resource.ManagedDirectory{
		BaseResource: resource.BaseResource{
			APIVersion: "dotisan/v1",
			Kind:       "ManagedDirectory",
			Metadata:   resource.Metadata{Name: "skills", Namespace: "default"},
		},
		Spec: resource.ManagedDirectorySpec{
			Source:      "skills",
			Destination: destSkillsDir,
			Recursive:   false,
			Clean:       false,
		},
	}

	// Reconcile with matching state
	desired := []resource.Resource{md}
	state := []provider.ResourceState{
		{
			ID:   "directory/default/skills",
			Kind: "ManagedDirectory",
			Name: "skills",
		},
	}
	plan := p.Reconcile(desired, state)

	// Should be in sync
	if len(plan.InSync) != 1 {
		t.Errorf("len(InSync) = %d, want 1", len(plan.InSync))
	}
}

func TestFileProvider_Reconcile_ManagedDirectory_Clean(t *testing.T) {
	// Setup temp directories
	dotfilesDir := t.TempDir()
	destDir := t.TempDir()

	// Create a source directory with one file
	sourceDir := filepath.Join(dotfilesDir, "skills")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "keep.txt"), []byte("keep"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Create destination with extra file
	destSkillsDir := filepath.Join(destDir, "skills")
	if err := os.MkdirAll(destSkillsDir, 0755); err != nil {
		t.Fatalf("Failed to create dest dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(destSkillsDir, "keep.txt"), []byte("keep"), 0644); err != nil {
		t.Fatalf("Failed to create dest file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(destSkillsDir, "remove.txt"), []byte("remove"), 0644); err != nil {
		t.Fatalf("Failed to create extra file: %v", err)
	}

	// Create provider
	p := NewFileProvider(nil, nil, dotfilesDir)

	// Create desired resource with clean: true
	md := &resource.ManagedDirectory{
		BaseResource: resource.BaseResource{
			APIVersion: "dotisan/v1",
			Kind:       "ManagedDirectory",
			Metadata:   resource.Metadata{Name: "skills", Namespace: "default"},
		},
		Spec: resource.ManagedDirectorySpec{
			Source:      "skills",
			Destination: destSkillsDir,
			Recursive:   false,
			Clean:       true, // Enable clean mode
		},
	}

	// Reconcile
	desired := []resource.Resource{md}
	state := []provider.ResourceState{
		{
			ID:   "directory/default/skills",
			Kind: "ManagedDirectory",
			Name: "skills",
		},
	}
	plan := p.Reconcile(desired, state)

	// Should have a modification (due to extra file that needs cleaning)
	if len(plan.Modifications) != 1 {
		t.Errorf("len(Modifications) = %d, want 1 (extra file should trigger change)", len(plan.Modifications))
	}
}

func TestFileProvider_shouldExclude(t *testing.T) {
	p := NewFileProvider(nil, nil, "")

	tests := []struct {
		path     string
		exclude  []string
		expected bool
	}{
		{"test.txt", []string{}, false},
		{"test.txt", []string{"*.txt"}, true},
		{"test.txt", []string{"*.go"}, false},
		{"dir/test.txt", []string{"*.txt"}, true},
		{"test.txt", []string{"test.txt"}, true},
		{"other.txt", []string{"test.txt"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := p.shouldExclude(tt.path, tt.exclude)
			if result != tt.expected {
				t.Errorf("shouldExclude(%q, %v) = %v, want %v", tt.path, tt.exclude, result, tt.expected)
			}
		})
	}
}

func TestFileProvider_walkSourceDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directory structure
	if err := os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "subdir", "file2.txt"), []byte("content2"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	p := NewFileProvider(nil, nil, "")

	// Test non-recursive
	files := make(map[string]string)
	if err := p.walkSourceDir(tmpDir, "", false, nil, files); err != nil {
		t.Fatalf("walkSourceDir failed: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("non-recursive walk found %d files, want 1", len(files))
	}

	// Test recursive
	files = make(map[string]string)
	if err := p.walkSourceDir(tmpDir, "", true, nil, files); err != nil {
		t.Fatalf("walkSourceDir failed: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("recursive walk found %d files, want 2", len(files))
	}
}
