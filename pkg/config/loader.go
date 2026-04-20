// Package config provides configuration loading and management for dotisan.
//
// This file provides a high-level loader that orchestrates the two-pass templating:
// 1. Load config.yaml (no templating needed)
// 2. Create TemplateContext with Env/OS
// 3. Load and render ~/.dotfiles/values.yaml
// 4. Return fully prepared context ready for resource file rendering
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Loader orchestrates the loading of all configuration files and preparation
// of the TemplateContext for resource rendering.
type Loader struct {
	// DotisanConfigPath is the path to ~/.dotisan/config.yaml
	DotisanConfigPath string

	// DotfilesRoot is the path to the dotfiles directory (e.g., ~/.dotfiles)
	DotfilesRoot string
}

// NewLoader creates a new Loader with default paths.
func NewLoader() (*Loader, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	return &Loader{
		DotisanConfigPath: filepath.Join(homeDir, ".dotisan", "config.yaml"),
		DotfilesRoot:      filepath.Join(homeDir, ".dotfiles"),
	}, nil
}

// NewLoaderWithPaths creates a new Loader with custom paths.
func NewLoaderWithPaths(dotisanConfigPath, dotfilesRoot string) *Loader {
	return &Loader{
		DotisanConfigPath: dotisanConfigPath,
		DotfilesRoot:      dotfilesRoot,
	}
}

// Load performs the complete configuration loading workflow:
// 1. Loads ~/.dotisan/config.yaml
// 2. Creates TemplateContext with Env/OS
// 3. Loads and renders ~/.dotfiles/values.yaml
// 4. Returns the prepared TemplateContext
func (l *Loader) Load() (*Config, *TemplateContext, error) {
	// Step 1: Load dotisan config
	cfg, err := LoadConfig(l.DotisanConfigPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Use dotfiles root from config if available, otherwise use loader default
	dotfilesRoot := cfg.DotfilesRoot
	if dotfilesRoot == "" {
		dotfilesRoot = l.DotfilesRoot
	}

	// Step 2: Create initial template context (Env + OS)
	ctx := NewTemplateContext()

	// Step 3: Load and render values.yaml (first pass)
	valuesPath := filepath.Join(dotfilesRoot, "values.yaml")
	engine := NewTemplateEngine(ctx)
	if err := engine.LoadAndRenderValues(valuesPath); err != nil {
		return nil, nil, fmt.Errorf("failed to load values: %w", err)
	}

	// Now ctx has .Values populated along with .Env and .OS
	// This context is ready for rendering resource files (second pass)

	return cfg, ctx, nil
}

// LoadComplete is a convenience function that performs the complete load
// using default paths.
func LoadComplete() (*Config, *TemplateContext, error) {
	loader, err := NewLoader()
	if err != nil {
		return nil, nil, err
	}
	return loader.Load()
}
