package resource

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wasilak/dotisan/pkg/config"
)

func TestNewLoader(t *testing.T) {
	ctx := config.NewTemplateContext()
	loader := NewLoader("/tmp/dotfiles", ctx)

	if loader == nil {
		t.Fatal("NewLoader() returned nil")
	}

	if loader.dotfilesPath != "/tmp/dotfiles" {
		t.Errorf("dotfilesPath = %q, want %q", loader.dotfilesPath, "/tmp/dotfiles")
	}

	if loader.engine == nil {
		t.Error("engine is nil")
	}
}

func TestLoader_LoadResources(t *testing.T) {
	// Create temp dotfiles structure
	tmpDir := t.TempDir()

	// Create values.yaml (should be skipped)
	valuesContent := "user:\n  name: testuser\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "values.yaml"), []byte(valuesContent), 0644); err != nil {
		t.Fatalf("Failed to write values.yaml: %v", err)
	}

	// Create a BrewPackages resource
	brewDir := filepath.Join(tmpDir, "brew")
	if err := os.MkdirAll(brewDir, 0755); err != nil {
		t.Fatalf("Failed to create brew dir: %v", err)
	}

	brewContent := "apiVersion: github.com/wasilak/dotisan/v1\nkind: BrewPackages\nmetadata:\n  name: core-tools\nspec:\n  formulae:\n    - name: ripgrep\n    - name: fd\n"
	if err := os.WriteFile(filepath.Join(brewDir, "core.yaml"), []byte(brewContent), 0644); err != nil {
		t.Fatalf("Failed to write brew resource: %v", err)
	}

	// Create a ManagedFile resource
	filesDir := filepath.Join(tmpDir, "files")
	if err := os.MkdirAll(filesDir, 0755); err != nil {
		t.Fatalf("Failed to create files dir: %v", err)
	}

	fileContent := "apiVersion: github.com/wasilak/dotisan/v1\nkind: ManagedFile\nmetadata:\n  name: test-config\nspec:\n  source: templates/test.txt\n  destination: /tmp/test.txt\n  template: false\n"
	if err := os.WriteFile(filepath.Join(filesDir, "test.yaml"), []byte(fileContent), 0644); err != nil {
		t.Fatalf("Failed to write file resource: %v", err)
	}

	// Create a non-YAML file (should be skipped)
	if err := os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# Dotfiles"), 0644); err != nil {
		t.Fatalf("Failed to write README: %v", err)
	}

	// Set up context with values
	ctx := config.NewTemplateContext()

	loader := NewLoader(tmpDir, ctx)
	resources, err := loader.LoadResources()
	if err != nil {
		t.Fatalf("LoadResources() failed: %v", err)
	}

	// Should have 2 resources (values.yaml and README.md should be skipped)
	if len(resources) != 2 {
		t.Errorf("LoadResources() returned %d resources, want 2", len(resources))
	}

	// Check types
	foundBrew := false
	foundFile := false
	for _, r := range resources {
		switch r.(type) {
		case *BrewPackages:
			foundBrew = true
			bp := r.(*BrewPackages)
			if bp.GetMetadata().Name != "core-tools" {
				t.Errorf("BrewPackages name = %q, want %q", bp.GetMetadata().Name, "core-tools")
			}
		case *ManagedFile:
			foundFile = true
			mf := r.(*ManagedFile)
			if mf.Spec.Source != "templates/test.txt" {
				t.Errorf("ManagedFile source = %q, want %q", mf.Spec.Source, "templates/test.txt")
			}
		}
	}

	if !foundBrew {
		t.Error("BrewPackages resource not found")
	}
	if !foundFile {
		t.Error("ManagedFile resource not found")
	}
}

func TestLoader_LoadResources_InvalidResource(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an invalid resource (unknown kind)
	invalidContent := "apiVersion: github.com/wasilak/dotisan/v1\nkind: UnknownKind\nmetadata:\n  name: invalid\nspec: {}\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "invalid.yaml"), []byte(invalidContent), 0644); err != nil {
		t.Fatalf("Failed to write invalid resource: %v", err)
	}

	ctx := config.NewTemplateContext()
	loader := NewLoader(tmpDir, ctx)

	_, err := loader.LoadResources()
	if err == nil {
		t.Error("LoadResources() should fail with invalid resource")
	}
}

func TestIsYAMLFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"test.yaml", true},
		{"test.yml", true},
		{"test.YAML", true},
		{"test.YML", true},
		{"test.json", false},
		{"test.txt", false},
		{"test", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := isYAMLFile(tt.path); got != tt.want {
				t.Errorf("isYAMLFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
