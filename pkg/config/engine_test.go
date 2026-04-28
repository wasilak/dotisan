package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewTemplateEngine(t *testing.T) {
	ctx := NewTemplateContext()
	engine := NewTemplateEngine(ctx)

	if engine == nil {
		t.Fatal("NewTemplateEngine() returned nil")
	}

	if engine.ctx != ctx {
		t.Error("TemplateEngine.ctx does not match provided context")
	}
}

func TestRenderTemplate_EnvInterpolation(t *testing.T) {
	ctx := NewTemplateContext()
	engine := NewTemplateEngine(ctx)

	template := "Home: {{ .Env.HOME }}"
	result, err := engine.RenderTemplate("test", template)
	if err != nil {
		t.Fatalf("RenderTemplate() failed: %v", err)
	}

	expectedHome := os.Getenv("HOME")
	expected := "Home: " + expectedHome
	if result != expected {
		t.Errorf("RenderTemplate() = %q, want %q", result, expected)
	}
}

func TestRenderTemplate_OSInfo(t *testing.T) {
	ctx := NewTemplateContext()
	engine := NewTemplateEngine(ctx)

	template := "OS: {{ .OS.GOOS }}, Arch: {{ .OS.GOARCH }}"
	result, err := engine.RenderTemplate("test", template)
	if err != nil {
		t.Fatalf("RenderTemplate() failed: %v", err)
	}

	// Verify result contains OS info
	if result == "" {
		t.Error("RenderTemplate() returned empty string")
	}
}

func TestRenderTemplate_SprigFunctions(t *testing.T) {
	ctx := NewTemplateContext()
	engine := NewTemplateEngine(ctx)

	// Test default function
	template := `{{ default "fallback" .Env.NONEXISTENT_VAR }}`
	result, err := engine.RenderTemplate("test", template)
	if err != nil {
		t.Fatalf("RenderTemplate() failed: %v", err)
	}

	if result != "fallback" {
		t.Errorf("RenderTemplate() = %q, want %q", result, "fallback")
	}
}

func TestRenderTemplate_SprigUpper(t *testing.T) {
	ctx := NewTemplateContext()
	engine := NewTemplateEngine(ctx)

	// Test upper function
	template := `{{ upper "hello" }}`
	result, err := engine.RenderTemplate("test", template)
	if err != nil {
		t.Fatalf("RenderTemplate() failed: %v", err)
	}

	if result != "HELLO" {
		t.Errorf("RenderTemplate() = %q, want %q", result, "HELLO")
	}
}

func TestLoadAndRenderValues_FileNotExists(t *testing.T) {
	ctx := NewTemplateContext()
	engine := NewTemplateEngine(ctx)

	// Use non-existent path
	nonExistentPath := "/tmp/values_does_not_exist.yaml"

	err := engine.LoadAndRenderValues(nonExistentPath)
	if err != nil {
		t.Fatalf("LoadAndRenderValues() with non-existent file should not error, got: %v", err)
	}

	// Values should remain empty (but initialized)
	if engine.ctx.Values == nil {
		t.Error("Values map should be initialized")
	}
}

func TestLoadAndRenderValues_ValidFile(t *testing.T) {
	// Set up test environment variable
	os.Setenv("DOTISAN_TEST_USER", "testuser")
	defer os.Unsetenv("DOTISAN_TEST_USER")

	ctx := NewTemplateContext()
	engine := NewTemplateEngine(ctx)

	// Create temporary values.yaml
	tmpDir := t.TempDir()
	valuesPath := filepath.Join(tmpDir, "values.yaml")

	valuesContent := `
user:
  name: "{{ .Env.DOTISAN_TEST_USER }}"
  home: "{{ .Env.HOME }}"
paths:
  dotfiles: "{{ .Env.HOME }}/.config/dotisan"
`

	if err := os.WriteFile(valuesPath, []byte(valuesContent), 0644); err != nil {
		t.Fatalf("Failed to write test values file: %v", err)
	}

	// Load and render
	err := engine.LoadAndRenderValues(valuesPath)
	if err != nil {
		t.Fatalf("LoadAndRenderValues() failed: %v", err)
	}

	// Verify values were parsed
	userMap, ok := engine.ctx.Values["user"].(map[string]interface{})
	if !ok {
		t.Fatal("Values.user is not a map")
	}

	if userMap["name"] != "testuser" {
		t.Errorf("user.name = %v, want %v", userMap["name"], "testuser")
	}

	homeEnv := os.Getenv("HOME")
	if userMap["home"] != homeEnv {
		t.Errorf("user.home = %v, want %v", userMap["home"], homeEnv)
	}
}

func TestLoadAndRenderValues_InvalidYAML(t *testing.T) {
	ctx := NewTemplateContext()
	engine := NewTemplateEngine(ctx)

	// Create temporary file with invalid YAML after rendering
	tmpDir := t.TempDir()
	valuesPath := filepath.Join(tmpDir, "values.yaml")

	// This template will render to invalid YAML
	valuesContent := `invalid: yaml: [{
`

	if err := os.WriteFile(valuesPath, []byte(valuesContent), 0644); err != nil {
		t.Fatalf("Failed to write test values file: %v", err)
	}

	err := engine.LoadAndRenderValues(valuesPath)
	if err == nil {
		t.Error("LoadAndRenderValues() should return error for invalid YAML")
	}
}
