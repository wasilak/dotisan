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

	// Create resources/ subdir for resource files
	resourcesDir := filepath.Join(tmpDir, "resources")
	if err := os.MkdirAll(resourcesDir, 0755); err != nil {
		t.Fatalf("Failed to create resources dir: %v", err)
	}

	// Create values.yaml (should be skipped)
	valuesContent := "user:\n  name: testuser\n"
	if err := os.WriteFile(filepath.Join(resourcesDir, "values.yaml"), []byte(valuesContent), 0644); err != nil {
		t.Fatalf("Failed to write values.yaml: %v", err)
	}

	// Create a BrewPackages resource
	brewDir := filepath.Join(resourcesDir, "brew")
	if err := os.MkdirAll(brewDir, 0755); err != nil {
		t.Fatalf("Failed to create brew dir: %v", err)
	}

	brewContent := "apiVersion: github.com/wasilak/dotisan/v1\nkind: HomeBrewPackages\nmetadata:\n  name: core-tools\nspec:\n  formulae:\n    - name: ripgrep\n    - name: fd\n"
	if err := os.WriteFile(filepath.Join(brewDir, "core.yaml"), []byte(brewContent), 0644); err != nil {
		t.Fatalf("Failed to write brew resource: %v", err)
	}

	// Create a ManagedFile resource
	filesDir := filepath.Join(resourcesDir, "files")
	if err := os.MkdirAll(filesDir, 0755); err != nil {
		t.Fatalf("Failed to create files dir: %v", err)
	}

	fileContent := "apiVersion: github.com/wasilak/dotisan/v1\nkind: ManagedFile\nmetadata:\n  name: test-config\nspec:\n  source: templates/test.txt\n  destination: /tmp/test.txt\n  template: false\n"
	if err := os.WriteFile(filepath.Join(filesDir, "test.yaml"), []byte(fileContent), 0644); err != nil {
		t.Fatalf("Failed to write file resource: %v", err)
	}

	// Create a non-YAML file (should be skipped)
	if err := os.WriteFile(filepath.Join(resourcesDir, "README.md"), []byte("# Dotfiles"), 0644); err != nil {
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
		case *HomeBrewPackages:
			foundBrew = true
			bp := r.(*HomeBrewPackages)
			if bp.GetMetadata().Name != "core-tools" {
				t.Errorf("HomeBrewPackages name = %q, want %q", bp.GetMetadata().Name, "core-tools")
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

	// Create resources/ subdir and put invalid resource there
	resourcesDir := filepath.Join(tmpDir, "resources")
	if err := os.MkdirAll(resourcesDir, 0755); err != nil {
		t.Fatalf("Failed to create resources dir: %v", err)
	}

	// Create an invalid resource (unknown kind)
	invalidContent := "apiVersion: github.com/wasilak/dotisan/v1\nkind: UnknownKind\nmetadata:\n  name: invalid\nspec: {}\n"
	if err := os.WriteFile(filepath.Join(resourcesDir, "invalid.yaml"), []byte(invalidContent), 0644); err != nil {
		t.Fatalf("Failed to write invalid resource: %v", err)
	}

	ctx := config.NewTemplateContext()
	loader := NewLoader(tmpDir, ctx)

	_, err := loader.LoadResources()
	if err == nil {
		t.Error("LoadResources() should fail with invalid resource")
	}
}

func TestLoader_LoadResources_WithGenerator(t *testing.T) {
	tmpDir := t.TempDir()
	resourcesDir := filepath.Join(tmpDir, "resources")
	if err := os.MkdirAll(resourcesDir, 0755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	generatorYAML := `apiVersion: github.com/wasilak/dotisan/v1
kind: ManagedFile
metadata:
  name: gen-skills
spec:
  generator:
    sourceKey: skills
    template: "skill: {{ .Item }}\nindex: {{ .Index }}"
    destinationPattern: "/tmp/skills/{{ .Item }}.md"
    mode: "0644"
`
	if err := os.WriteFile(filepath.Join(resourcesDir, "skills.yaml"), []byte(generatorYAML), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	ctx := config.NewTemplateContext()
	ctx.Values = map[string]any{
		"skills": []any{"bash", "python", "go"},
	}

	loader := NewLoader(tmpDir, ctx)
	resources, err := loader.LoadResources()
	if err != nil {
		t.Fatalf("LoadResources() failed: %v", err)
	}

	if len(resources) != 1 {
		t.Fatalf("want 1 resource, got %d", len(resources))
	}

	mf, ok := resources[0].(*ManagedFile)
	if !ok {
		t.Fatalf("expected *ManagedFile, got %T", resources[0])
	}

	// Generator should have been cleared
	if mf.Spec.Generator != nil {
		t.Error("Spec.Generator should be nil after expansion")
	}

	// Should have 3 FileSpecs
	if len(mf.Spec.Files) != 3 {
		t.Fatalf("want 3 files, got %d", len(mf.Spec.Files))
	}

	tests := []struct {
		wantSource string
		wantDest   string
		wantMode   string
	}{
		{"skill: bash\nindex: 0", "/tmp/skills/bash.md", "0644"},
		{"skill: python\nindex: 1", "/tmp/skills/python.md", "0644"},
		{"skill: go\nindex: 2", "/tmp/skills/go.md", "0644"},
	}

	for i, tt := range tests {
		f := mf.Spec.Files[i]
		if f.Source != tt.wantSource {
			t.Errorf("file[%d].Source = %q, want %q", i, f.Source, tt.wantSource)
		}
		if f.Destination != tt.wantDest {
			t.Errorf("file[%d].Destination = %q, want %q", i, f.Destination, tt.wantDest)
		}
		if f.Mode != tt.wantMode {
			t.Errorf("file[%d].Mode = %q, want %q", i, f.Mode, tt.wantMode)
		}
	}
}

func TestLoader_LoadResources_GeneratorMapItems(t *testing.T) {
	tmpDir := t.TempDir()
	resourcesDir := filepath.Join(tmpDir, "resources")
	if err := os.MkdirAll(resourcesDir, 0755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	generatorYAML := `apiVersion: github.com/wasilak/dotisan/v1
kind: ManagedFile
metadata:
  name: gen-agents
spec:
  generator:
    sourceKey: agents
    template: "name: {{ .Item.name }}\ndesc: {{ .Item.desc }}"
    destinationPattern: "/tmp/agents/{{ .Item.name }}.md"
`
	if err := os.WriteFile(filepath.Join(resourcesDir, "agents.yaml"), []byte(generatorYAML), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	ctx := config.NewTemplateContext()
	ctx.Values = map[string]any{
		"agents": []any{
			map[string]any{"name": "coder", "desc": "writes code"},
			map[string]any{"name": "reviewer", "desc": "reviews code"},
		},
	}

	loader := NewLoader(tmpDir, ctx)
	resources, err := loader.LoadResources()
	if err != nil {
		t.Fatalf("LoadResources() failed: %v", err)
	}

	mf := resources[0].(*ManagedFile)
	if len(mf.Spec.Files) != 2 {
		t.Fatalf("want 2 files, got %d", len(mf.Spec.Files))
	}
	if mf.Spec.Files[0].Source != "name: coder\ndesc: writes code" {
		t.Errorf("file[0].Source = %q", mf.Spec.Files[0].Source)
	}
	if mf.Spec.Files[0].Destination != "/tmp/agents/coder.md" {
		t.Errorf("file[0].Destination = %q", mf.Spec.Files[0].Destination)
	}
}

func TestLoader_LoadResources_GeneratorEmptyList(t *testing.T) {
	tmpDir := t.TempDir()
	resourcesDir := filepath.Join(tmpDir, "resources")
	if err := os.MkdirAll(resourcesDir, 0755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	generatorYAML := `apiVersion: github.com/wasilak/dotisan/v1
kind: ManagedFile
metadata:
  name: gen-empty
spec:
  generator:
    sourceKey: empty
    template: "{{ .Item }}"
    destinationPattern: "/tmp/{{ .Item }}.md"
`
	if err := os.WriteFile(filepath.Join(resourcesDir, "empty.yaml"), []byte(generatorYAML), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	ctx := config.NewTemplateContext()
	ctx.Values = map[string]any{
		"empty": []any{},
	}

	loader := NewLoader(tmpDir, ctx)
	resources, err := loader.LoadResources()
	if err != nil {
		t.Fatalf("LoadResources() should succeed with empty list: %v", err)
	}

	mf := resources[0].(*ManagedFile)
	if len(mf.Spec.Files) != 0 {
		t.Errorf("want 0 files for empty list, got %d", len(mf.Spec.Files))
	}
}

func TestLoader_LoadResources_GeneratorInvalidSourceKey(t *testing.T) {
	tmpDir := t.TempDir()
	resourcesDir := filepath.Join(tmpDir, "resources")
	if err := os.MkdirAll(resourcesDir, 0755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	generatorYAML := `apiVersion: github.com/wasilak/dotisan/v1
kind: ManagedFile
metadata:
  name: gen-bad-key
spec:
  generator:
    sourceKey: notexist
    template: "{{ .Item }}"
    destinationPattern: "/tmp/{{ .Item }}.md"
`
	if err := os.WriteFile(filepath.Join(resourcesDir, "bad.yaml"), []byte(generatorYAML), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	ctx := config.NewTemplateContext()
	loader := NewLoader(tmpDir, ctx)

	_, err := loader.LoadResources()
	if err == nil {
		t.Error("expected error for missing sourceKey, got nil")
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
