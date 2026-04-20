package providers

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/resource"
)

func TestFileProvider_Apply_Directory_Addition(t *testing.T) {
	// Setup temp directories
	dotfilesDir := t.TempDir()
	destDir := t.TempDir()

	// Create resources subdirectory (dirs are resolved relative to resources/)
	resourcesDir := filepath.Join(dotfilesDir, "resources")
	os.MkdirAll(resourcesDir, 0755)

	// Create source directory with files in resources/
	sourceDir := filepath.Join(resourcesDir, "skills")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "skill1.txt"), []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "skill2.txt"), []byte("content2"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Create provider
	p := NewFileProvider(nil, nil, dotfilesDir)

	// Create plan with directory addition
	md := &resource.ManagedDirectory{
		BaseResource: resource.BaseResource{
			APIVersion: "github.com/wasilak/dotisan/v1",
			Kind:       "ManagedDirectory",
			Metadata:   resource.Metadata{Name: "skills", Namespace: "default"},
		},
		Spec: resource.ManagedDirectorySpec{
			SourceDir:   "skills",
			Destination: filepath.Join(destDir, "skills"),
			Recursive:   false,
			Clean:       false,
		},
	}

	plan := provider.Plan{
		Additions: []resource.Resource{md},
	}

	// Apply
	ctx := context.Background()
	if err := p.Apply(ctx, plan); err != nil {
		t.Fatalf("Apply() failed: %v", err)
	}

	// Verify directory and files were created
	destSkillsDir := filepath.Join(destDir, "skills")
	if _, err := os.Stat(destSkillsDir); os.IsNotExist(err) {
		t.Error("destination directory was not created")
	}

	// Verify files were copied
	for _, filename := range []string{"skill1.txt", "skill2.txt"} {
		destFile := filepath.Join(destSkillsDir, filename)
		if _, err := os.Stat(destFile); os.IsNotExist(err) {
			t.Errorf("destination file %s was not created", filename)
		}
	}
}

func TestFileProvider_Apply_Directory_Removal(t *testing.T) {
	// Setup temp directories
	destDir := t.TempDir()

	// Create destination directory with files
	destSkillsDir := filepath.Join(destDir, "skills")
	if err := os.MkdirAll(destSkillsDir, 0755); err != nil {
		t.Fatalf("Failed to create dest dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(destSkillsDir, "skill.txt"), []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Create provider
	p := NewFileProvider(nil, nil, "")

	// Create plan with directory removal
	md := &resource.ManagedDirectory{
		BaseResource: resource.BaseResource{
			APIVersion: "github.com/wasilak/dotisan/v1",
			Kind:       "ManagedDirectory",
			Metadata:   resource.Metadata{Name: "skills", Namespace: "default"},
		},
		Spec: resource.ManagedDirectorySpec{
			SourceDir:   "skills",
			Destination: destSkillsDir,
			Recursive:   false,
			Clean:       false,
		},
	}

	plan := provider.Plan{
		Removals: []resource.Resource{md},
	}

	// Apply
	ctx := context.Background()
	if err := p.Apply(ctx, plan); err != nil {
		t.Fatalf("Apply() failed: %v", err)
	}

	// Verify directory was removed
	if _, err := os.Stat(destSkillsDir); !os.IsNotExist(err) {
		t.Error("destination directory should have been removed")
	}
}

func TestFileProvider_Apply_Directory_Clean(t *testing.T) {
	// Setup temp directories
	dotfilesDir := t.TempDir()
	destDir := t.TempDir()

	// Create resources subdirectory (dirs are resolved relative to resources/)
	resourcesDir := filepath.Join(dotfilesDir, "resources")
	os.MkdirAll(resourcesDir, 0755)

	// Create source directory with one file in resources/
	sourceDir := filepath.Join(resourcesDir, "skills")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "keep.txt"), []byte("keep"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Create destination with matching file and extra file
	destSkillsDir := filepath.Join(destDir, "skills")
	if err := os.MkdirAll(destSkillsDir, 0755); err != nil {
		t.Fatalf("Failed to create dest dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(destSkillsDir, "keep.txt"), []byte("keep"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(destSkillsDir, "remove.txt"), []byte("remove"), 0644); err != nil {
		t.Fatalf("Failed to create extra file: %v", err)
	}

	// Create provider
	p := NewFileProvider(nil, nil, dotfilesDir)

	// Create plan with directory modification and clean
	md := &resource.ManagedDirectory{
		BaseResource: resource.BaseResource{
			APIVersion: "github.com/wasilak/dotisan/v1",
			Kind:       "ManagedDirectory",
			Metadata:   resource.Metadata{Name: "skills", Namespace: "default"},
		},
		Spec: resource.ManagedDirectorySpec{
			SourceDir:   "skills",
			Destination: destSkillsDir,
			Recursive:   false,
			Clean:       true, // Enable clean
		},
	}

	plan := provider.Plan{
		Modifications: []provider.Modification{
			{
				Resource: md,
			},
		},
	}

	// Apply
	ctx := context.Background()
	if err := p.Apply(ctx, plan); err != nil {
		t.Fatalf("Apply() failed: %v", err)
	}

	// Verify keep.txt still exists
	keepFile := filepath.Join(destSkillsDir, "keep.txt")
	if _, err := os.Stat(keepFile); os.IsNotExist(err) {
		t.Error("keep.txt should still exist")
	}

	// Verify remove.txt was removed
	removeFile := filepath.Join(destSkillsDir, "remove.txt")
	if _, err := os.Stat(removeFile); !os.IsNotExist(err) {
		t.Error("remove.txt should have been removed by clean")
	}
}

func TestFileProvider_syncDirectory(t *testing.T) {
	// Setup temp directories
	sourceDir := t.TempDir()
	destDir := t.TempDir()

	// Create source files
	if err := os.WriteFile(filepath.Join(sourceDir, "file1.txt"), []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	subDir := filepath.Join(sourceDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "file2.txt"), []byte("content2"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	// Create provider
	p := NewFileProvider(nil, nil, "")

	// Sync directory recursively
	ctx := context.Background()
	if err := p.syncDirectory(ctx, sourceDir, destDir, true, nil, false); err != nil {
		t.Fatalf("syncDirectory() failed: %v", err)
	}

	// Verify files were copied
	destFile1 := filepath.Join(destDir, "file1.txt")
	if _, err := os.Stat(destFile1); os.IsNotExist(err) {
		t.Error("file1.txt was not copied")
	}

	destFile2 := filepath.Join(destDir, "subdir", "file2.txt")
	if _, err := os.Stat(destFile2); os.IsNotExist(err) {
		t.Error("subdir/file2.txt was not copied")
	}
}
