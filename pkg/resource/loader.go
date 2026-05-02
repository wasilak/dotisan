package resource

import (
	"fmt"
	"log/slog"
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

	slog.Debug("rendered resource",
		"path", path,
		"content", rendered,
	)

	// Parse into resource
	resource, err := UnmarshalYAML([]byte(rendered))
	if err != nil {
		return nil, fmt.Errorf("failed to parse resource: %w", err)
	}

	// Validate the resource
	if err := resource.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Resolve relative sourceFile paths relative to the resource file's directory.
	resolveSourceFilePaths(resource, filepath.Dir(path))

	return resource, nil
}

// resolveSourceFilePaths makes relative sourceFile paths absolute by joining
// them with the directory of the resource file that declared them.
// Absolute paths and inline sources are left unchanged.
func resolveSourceFilePaths(res Resource, dir string) {
	mf, ok := res.(*ManagedFile)
	if !ok {
		return
	}
	if mf.Spec.SourceFile != "" && !filepath.IsAbs(mf.Spec.SourceFile) {
		mf.Spec.SourceFile = filepath.Join(dir, mf.Spec.SourceFile)
	}
	for i, f := range mf.Spec.Files {
		if f.SourceFile != "" && !filepath.IsAbs(f.SourceFile) {
			mf.Spec.Files[i].SourceFile = filepath.Join(dir, f.SourceFile)
		}
	}
}

// isYAMLFile checks if a path has a YAML extension.
func isYAMLFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yaml" || ext == ".yml"
}
