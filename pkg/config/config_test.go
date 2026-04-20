package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	// Check that DotfilesRoot is set to a non-empty value
	if cfg.DotfilesRoot == "" {
		t.Error("DefaultConfig().DotfilesRoot is empty")
	}

	// Check default state backend
	if cfg.State.Backend != "local" {
		t.Errorf("DefaultConfig().State.Backend = %q, want %q", cfg.State.Backend, "local")
	}

	// Check default state path is non-empty
	if cfg.State.Path == "" {
		t.Error("DefaultConfig().State.Path is empty")
	}
}

func TestLoadConfig_FileNotExists(t *testing.T) {
	// Use a path that doesn't exist
	nonExistentPath := "/tmp/dotisan_config_does_not_exist.yaml"

	cfg, err := LoadConfig(nonExistentPath)
	if err != nil {
		t.Fatalf("LoadConfig() with non-existent file should return defaults, got error: %v", err)
	}

	if cfg == nil {
		t.Fatal("LoadConfig() returned nil config")
	}

	// Verify we got defaults
	if cfg.DotfilesRoot == "" {
		t.Error("LoadConfig() returned empty DotfilesRoot")
	}
}

func TestLoadConfig_ValidFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
dotfiles_root: /custom/path/to/dotfiles
state:
  backend: s3
  path: /some/local/path.json
  s3:
    endpoint: s3.amazonaws.com
    bucket: my-bucket
    key: state.json
    region: us-east-1
    access_key_id: AKIAIOSFODNN7EXAMPLE
    secret_access_key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	// Verify loaded values
	if cfg.DotfilesRoot != "/custom/path/to/dotfiles" {
		t.Errorf("DotfilesRoot = %q, want %q", cfg.DotfilesRoot, "/custom/path/to/dotfiles")
	}

	if cfg.State.Backend != "s3" {
		t.Errorf("State.Backend = %q, want %q", cfg.State.Backend, "s3")
	}

	if cfg.State.S3.Bucket != "my-bucket" {
		t.Errorf("State.S3.Bucket = %q, want %q", cfg.State.S3.Bucket, "my-bucket")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	// Create a temporary file with invalid YAML
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	invalidContent := `
dotfiles_root: [invalid: yaml: content
`

	if err := os.WriteFile(configPath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("LoadConfig() should return error for invalid YAML")
	}
}
