package resource

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wasilak/dotisan/pkg/config"
)

// Loader handles loading resources from YAML files in the dotfiles directory.
type Loader struct {
	dotfilesPath string
	engine       *config.TemplateEngine
}

// NewLoader creates a new resource loader.
func NewLoader(dotfilesPath string, ctx *config.TemplateContext) *Loader {
	return &Loader{
		dotfilesPath: dotfilesPath,
		engine:       config.NewTemplateEngine(ctx),
	}
}

// LoadResources recursively scans the dotfiles directory for YAML resource files
// and loads them into Resource objects.
func (l *Loader) LoadResources() ([]Resource, error) {
	var resources []Resource

	err := filepath.Walk(l.dotfilesPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process YAML files
		if !isYAMLFile(path) {
			return nil
		}

		// Skip values.yaml and config.yaml (tool-level configs)
		base := filepath.Base(path)
		if base == "values.yaml" || base == "config.yaml" {
			return nil
		}

		// Skip sample/example files
		if strings.HasPrefix(base, "sample") || strings.HasPrefix(base, "test-") {
			return nil
		}

		// Load and parse the resource file
		resource, err := l.loadResourceFile(path)
		if err != nil {
			return fmt.Errorf("failed to load %s: %w", path, err)
		}

		if resource != nil {
			resources = append(resources, resource)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk dotfiles directory: %w", err)
	}

	return resources, nil
}

// loadResourceFile loads a single YAML resource file.
func (l *Loader) loadResourceFile(path string) (Resource, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Render as template (second pass - uses full context including .Values)
	rendered, err := l.engine.RenderTemplate(path, string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to render template: %w", err)
	}

	// Parse into resource
	resource, err := UnmarshalYAML([]byte(rendered))
	if err != nil {
		return nil, fmt.Errorf("failed to parse resource: %w", err)
	}

	// Validate the resource
	if err := resource.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return resource, nil
}

// isYAMLFile checks if a path has a YAML extension.
func isYAMLFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yaml" || ext == ".yml"
}
