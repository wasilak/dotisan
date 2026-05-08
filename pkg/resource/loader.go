package resource

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/wasilak/nim/pkg/config"
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

	// Only scan the 'resources' subdirectory inside the dotfilesPath. This
	// ensures generated files at the dotfiles root are ignored.
	resourcesRoot := filepath.Join(l.dotfilesPath, "resources")
	if _, err := os.Stat(resourcesRoot); os.IsNotExist(err) {
		// No resources directory -> nothing to load
		return resources, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to access resources directory: %w", err)
	}

	err := filepath.Walk(resourcesRoot, func(path string, info os.FileInfo, err error) error {
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
		return nil, fmt.Errorf("failed to walk resources directory: %w", err)
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

	// Shield generator template fields from the first-pass resource-file render:
	// replace {{ .Item }}/{{ .Index }} expressions with placeholders so they survive
	// TemplateContext rendering, then restore originals after parsing.
	shielded, rawGenTmpls, err := shieldGeneratorTemplates(data)
	if err != nil {
		return nil, fmt.Errorf("failed to shield generator templates: %w", err)
	}

	// Render as template (second pass - uses full context including .Values)
	rendered, err := l.engine.RenderTemplate(path, string(shielded))
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

	// Restore raw generator template strings (first-pass render may have corrupted them).
	if mf, ok := resource.(*ManagedFile); ok {
		restoreRawGeneratorTemplates(mf, rawGenTmpls)
	}

	// Validate the resource
	if err := resource.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Expand generator specs into concrete FileSpec entries before resolving source files.
	if mf, ok := resource.(*ManagedFile); ok {
		if err := expandGenerator(mf, l.engine.Context(), filepath.Dir(path)); err != nil {
			return nil, fmt.Errorf("generator expansion failed: %w", err)
		}
	}

	// Resolve relative sourceFile paths and render templates where requested.
	if err := l.resolveSourceFiles(resource, filepath.Dir(path)); err != nil {
		return nil, err
	}

	return resource, nil
}

// resolveSourceFiles handles sourceFile fields on ManagedFile resources:
//   - Relative paths are made absolute relative to the resource file's directory.
//   - When template: true the file content is rendered through the template engine
//     and stored as inline Source, so downstream code sees pre-rendered content.
func (l *Loader) resolveSourceFiles(res Resource, dir string) error {
	mf, ok := res.(*ManagedFile)
	if !ok {
		return nil
	}

	rendered, err := l.renderSourceFile(mf.Spec.SourceFile, mf.Spec.Template, dir, mf.Spec.Vars)
	if err != nil {
		return err
	}
	if rendered != nil {
		mf.Spec.Source = *rendered
		mf.Spec.SourceFile = ""
	} else if mf.Spec.SourceFile != "" && !filepath.IsAbs(mf.Spec.SourceFile) {
		mf.Spec.SourceFile = filepath.Join(dir, mf.Spec.SourceFile)
	}

	for i, f := range mf.Spec.Files {
		rendered, err := l.renderSourceFile(f.SourceFile, f.Template, dir, f.Vars)
		if err != nil {
			return err
		}
		if rendered != nil {
			mf.Spec.Files[i].Source = *rendered
			mf.Spec.Files[i].SourceFile = ""
		} else if f.SourceFile != "" && !filepath.IsAbs(f.SourceFile) {
			mf.Spec.Files[i].SourceFile = filepath.Join(dir, f.SourceFile)
		}
	}

	return nil
}

// renderSourceFile reads and optionally renders a sourceFile as a template.
// Returns nil when sourceFile is empty or template is false (no rendering needed).
// Returns a pointer to the rendered string when template rendering was performed.
// When vars is non-empty, templates can access manifest-level variables as .Vars.*.
func (l *Loader) renderSourceFile(sourceFile string, isTemplate bool, dir string, vars map[string]any) (*string, error) {
	if sourceFile == "" || !isTemplate {
		return nil, nil
	}

	path := sourceFile
	if !filepath.IsAbs(path) {
		path = filepath.Join(dir, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read source file %s: %w", path, err)
	}

	var result string
	if len(vars) > 0 {
		result, err = l.engine.RenderTemplateWithVars(path, string(data), vars)
	} else {
		result, err = l.engine.RenderTemplate(path, string(data))
	}
	if err != nil {
		return nil, fmt.Errorf("failed to render template %s: %w", path, err)
	}

	return &result, nil
}

// isYAMLFile checks if a path has a YAML extension.
func isYAMLFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yaml" || ext == ".yml"
}
