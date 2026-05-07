package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewLoader(t *testing.T) {
	loader, err := NewLoader()
	if err != nil {
		t.Fatalf("NewLoader() failed: %v", err)
	}

	if loader == nil {
		t.Fatal("NewLoader() returned nil")
	}

	// Verify default paths are set
	if loader.DotisanConfigPath == "" {
		t.Error("DotisanConfigPath is empty")
	}

	if loader.DotfilesRoot == "" {
		t.Error("DotfilesRoot is empty")
	}

	// Verify paths contain expected components
	if !contains(loader.DotisanConfigPath, ".dotisan") {
		t.Errorf("DotisanConfigPath %q does not contain '.dotisan'", loader.DotisanConfigPath)
	}

	if !contains(loader.DotfilesRoot, ".config/dotisan") {
		t.Errorf("DotfilesRoot %q does not contain '.config/dotisan'", loader.DotfilesRoot)
	}
}

func TestNewLoaderWithPaths(t *testing.T) {
	loader := NewLoaderWithPaths("/custom/config.yaml", "/custom/dotfiles")

	if loader.DotisanConfigPath != "/custom/config.yaml" {
		t.Errorf("DotisanConfigPath = %q, want %q", loader.DotisanConfigPath, "/custom/config.yaml")
	}

	if loader.DotfilesRoot != "/custom/dotfiles" {
		t.Errorf("DotfilesRoot = %q, want %q", loader.DotfilesRoot, "/custom/dotfiles")
	}
}

func TestLoader_Load(t *testing.T) {
	// Create temporary directories and files
	tmpDir := t.TempDir()
	dotisanDir := filepath.Join(tmpDir, ".dotisan")
	dotfilesDir := filepath.Join(tmpDir, ".config/dotisan")

	if err := os.MkdirAll(dotisanDir, 0755); err != nil {
		t.Fatalf("Failed to create dotisan dir: %v", err)
	}
	if err := os.MkdirAll(dotfilesDir, 0755); err != nil {
		t.Fatalf("Failed to create dotfiles dir: %v", err)
	}

	// Create config.yaml
	configContent := `
dotfiles_root: ` + dotfilesDir + `
state:
  backend: local
  path: /tmp/test-state.json
`
	configPath := filepath.Join(dotisanDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Create values.yaml with template
	valuesContent := `
user:
  name: "{{ .Env.USER }}"
  home: "{{ .Env.HOME }}"
`
	valuesPath := filepath.Join(dotfilesDir, "values.yaml")
	if err := os.WriteFile(valuesPath, []byte(valuesContent), 0644); err != nil {
		t.Fatalf("Failed to write values: %v", err)
	}

	// Load using custom paths
	loader := NewLoaderWithPaths(configPath, dotfilesDir)
	cfg, ctx, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify config loaded
	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}

	if cfg.State.Backend != "local" {
		t.Errorf("State.Backend = %q, want %q", cfg.State.Backend, "local")
	}

	// Verify template context has values
	if ctx == nil {
		t.Fatal("Load() returned nil context")
	}

	if ctx.Values == nil {
		t.Error("TemplateContext.Values is nil")
	}

	// Check that values were parsed
	userMap, ok := ctx.Values["user"].(map[string]any)
	if !ok {
		t.Fatal("Values.user is not a map")
	}

	// Check environment-based values were rendered
	if userMap["name"] == "" {
		t.Error("user.name was not rendered from template")
	}

	if userMap["home"] == "" {
		t.Error("user.home was not rendered from template")
	}

	// Verify Env and OS are populated
	if ctx.Env == nil {
		t.Error("TemplateContext.Env is nil")
	}

	if ctx.Env["HOME"] == "" {
		t.Error("HOME not in Env")
	}

	if ctx.OS.GOOS == "" {
		t.Error("OS.GOOS is empty")
	}
}

func TestLoader_Load_NoValuesFile(t *testing.T) {
	// Create temporary directories without values.yaml
	tmpDir := t.TempDir()
	dotisanDir := filepath.Join(tmpDir, ".dotisan")
	dotfilesDir := filepath.Join(tmpDir, ".config/dotisan")

	if err := os.MkdirAll(dotisanDir, 0755); err != nil {
		t.Fatalf("Failed to create dotisan dir: %v", err)
	}
	if err := os.MkdirAll(dotfilesDir, 0755); err != nil {
		t.Fatalf("Failed to create dotfiles dir: %v", err)
	}

	// Create config.yaml only
	configContent := `state:
  backend: local
`
	configPath := filepath.Join(dotisanDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Load without values.yaml
	loader := NewLoaderWithPaths(configPath, dotfilesDir)
	cfg, ctx, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Should succeed with empty values
	if ctx.Values == nil {
		t.Error("Values should be initialized even without values.yaml")
	}

	// Config should still be loaded
	if cfg == nil {
		t.Error("Config should be loaded")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
