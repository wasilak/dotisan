package resource

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/wasilak/dotisan/pkg/config"
	"gopkg.in/yaml.v3"
)

// generatorContext is the template data available when rendering generator templates.
// Fields use PascalCase so they are accessible in templates as .Item, .Index, etc.
type generatorContext struct {
	Item   interface{}
	Index  int
	Values map[string]interface{}
	Env    map[string]string
	OS     config.OSInfo
}

// renderGeneratorTemplate renders a template string with the given per-item context.
// name is used only for error messages (e.g. resource name + field).
// dir is the base directory for resolving relative paths in readFile calls.
func renderGeneratorTemplate(name, tmplStr string, ctx generatorContext, dir string) (string, error) {
	readFile := func(path string) (string, error) {
		if !strings.HasPrefix(path, "/") {
			path = dir + "/" + path
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("readFile %q: %w", path, err)
		}
		return string(data), nil
	}

	tmpl, err := template.New(name).
		Funcs(sprig.TxtFuncMap()).
		Funcs(template.FuncMap{"readFile": readFile}).
		Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("template parse error in %s: %w", name, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return "", fmt.Errorf("template render error in %s (item index %d): %w", name, ctx.Index, err)
	}

	return buf.String(), nil
}

// expandGenerator resolves a ManagedFile's GeneratorSpec into concrete FileSpec entries in-memory.
// dir is the directory of the resource YAML file, used to resolve relative sourceFilePattern paths.
// After expansion, Spec.Files is populated with the generated entries and Spec.Generator is cleared,
// so downstream code (engine, providers) sees a normal ManagedFile with no special handling required.
// Returns nil immediately when mf has no GeneratorSpec.
func expandGenerator(mf *ManagedFile, ctx *config.TemplateContext, dir string) error {
	gen := mf.Spec.Generator
	if gen == nil {
		return nil
	}

	items, err := resolveSourceKey(gen.SourceKey, ctx.Values)
	if err != nil {
		return fmt.Errorf("resource %q: %w", mf.Metadata.Name, err)
	}

	files := make([]FileSpec, 0, len(items))

	for i, item := range items {
		tctx := generatorContext{
			Item:   item,
			Index:  i,
			Values: ctx.Values,
			Env:    ctx.Env,
			OS:     ctx.OS,
		}

		content, err := resolveGeneratorContent(mf.Metadata.Name, i, gen, tctx, dir)
		if err != nil {
			return err
		}

		destName := fmt.Sprintf("%s[%d].destinationPattern", mf.Metadata.Name, i)
		dest, err := renderGeneratorTemplate(destName, gen.DestinationPattern, tctx, dir)
		if err != nil {
			return err
		}

		if strings.TrimSpace(dest) == "" {
			return fmt.Errorf("resource %q: destinationPattern rendered to empty string at index %d", mf.Metadata.Name, i)
		}

		files = append(files, FileSpec{
			Source:      content,
			Destination: dest,
			Mode:        gen.Mode,
			DependsOn:   gen.DependsOn,
		})
	}

	mf.Spec.Files = files
	mf.Spec.Generator = nil

	return nil
}

// resolveGeneratorContent produces the file content for one generator item.
// If gen.Template is set, renders it as a Go template.
// If gen.SourceFilePattern is set, renders the pattern to get a path and reads the file raw.
func resolveGeneratorContent(resourceName string, index int, gen *GeneratorSpec, tctx generatorContext, dir string) (string, error) {
	if gen.Template != "" {
		name := fmt.Sprintf("%s[%d].template", resourceName, index)
		return renderGeneratorTemplate(name, gen.Template, tctx, dir)
	}

	// SourceFilePattern: render pattern → path → read file bytes
	name := fmt.Sprintf("%s[%d].sourceFilePattern", resourceName, index)
	relPath, err := renderGeneratorTemplate(name, gen.SourceFilePattern, tctx, dir)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(relPath) == "" {
		return "", fmt.Errorf("resource %q: sourceFilePattern rendered to empty string at index %d", resourceName, index)
	}

	absPath := relPath
	if !strings.HasPrefix(relPath, "/") {
		absPath = dir + "/" + relPath
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("resource %q: sourceFilePattern: failed to read %s at index %d: %w", resourceName, absPath, index, err)
	}

	return string(data), nil
}

// expandTilde replaces a leading "~" with the user's home directory.
func expandTilde(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return home + path[1:]
}

// resolveSourceKey resolves a dot-notation path into values and returns the list it points to.
// Only simple dot-notation is supported (e.g. "skills" or "agents.skills"); array indexing is not.
// Returns an error if the path is missing, intermediate nodes are not maps, or the final value is not a list.
func resolveSourceKey(key string, values map[string]interface{}) ([]interface{}, error) {
	if key == "" {
		return nil, fmt.Errorf("sourceKey must not be empty")
	}

	parts := strings.Split(key, ".")
	var current interface{} = values

	for i, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			path := strings.Join(parts[:i], ".")
			return nil, fmt.Errorf("sourceKey %q: %q is not a map", key, path)
		}

		val, exists := m[part]
		if !exists {
			return nil, fmt.Errorf("sourceKey %q: key %q not found", key, part)
		}

		current = val
	}

	slice, ok := current.([]interface{})
	if !ok {
		return nil, fmt.Errorf("sourceKey %q: value is not a list (got %T)", key, current)
	}

	return slice, nil
}

// rawGeneratorTemplates holds the original (un-rendered) generator template strings
// extracted from a resource YAML before first-pass template rendering.
type rawGeneratorTemplates struct {
	template           string
	destinationPattern string
}

// shieldGeneratorTemplates parses raw YAML, extracts generator template/destinationPattern
// strings, replaces them with safe placeholder text, and returns the modified YAML bytes
// alongside the originals. This prevents generator-level {{ .Item }}/{{ .Index }} expressions
// from being evaluated during the resource-file first-pass template render (which uses
// TemplateContext, not generatorContext).
// Returns the original data unchanged when no generator spec is found.
func shieldGeneratorTemplates(data []byte) (shielded []byte, raw *rawGeneratorTemplates, err error) {
	// Quick check: skip the YAML parse if there's no generator key at all.
	if !strings.Contains(string(data), "generator:") {
		return data, nil, nil
	}

	var doc map[string]interface{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		// Let the normal render path surface the parse error.
		return data, nil, nil
	}

	spec, _ := doc["spec"].(map[string]interface{})
	if spec == nil {
		return data, nil, nil
	}

	gen, _ := spec["generator"].(map[string]interface{})
	if gen == nil {
		return data, nil, nil
	}

	raw = &rawGeneratorTemplates{
		template:           fmt.Sprintf("%v", gen["template"]),
		destinationPattern: fmt.Sprintf("%v", gen["destinationPattern"]),
	}

	// Replace with innocuous placeholders that survive Go template rendering.
	gen["template"] = "__generator_template_placeholder__"
	gen["destinationPattern"] = "__generator_destinationPattern_placeholder__"

	shieldedBytes, err := yaml.Marshal(doc)
	if err != nil {
		return data, nil, nil
	}
	return shieldedBytes, raw, nil
}

// restoreRawGeneratorTemplates writes the preserved raw template strings back onto
// the parsed ManagedFile, overwriting whatever the first-pass render produced.
func restoreRawGeneratorTemplates(mf *ManagedFile, raw *rawGeneratorTemplates) {
	if mf.Spec.Generator == nil || raw == nil {
		return
	}
	mf.Spec.Generator.Template = raw.template
	mf.Spec.Generator.DestinationPattern = raw.destinationPattern
}
