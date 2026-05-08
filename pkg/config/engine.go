// Package config provides configuration loading and management for dotisan.
//
// This file contains the template engine for two-pass rendering.
// First pass: render values.yaml using Env/OS context.
// Second pass: render resource files using full context including .Values.
package config

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"gopkg.in/yaml.v3"
)

// TemplateEngine provides Go template rendering with Sprig functions.
type TemplateEngine struct {
	ctx *TemplateContext
}

// NewTemplateEngine creates a new template engine with the given context.
func NewTemplateEngine(ctx *TemplateContext) *TemplateEngine {
	return &TemplateEngine{ctx: ctx}
}

// Context returns the template context used by this engine.
func (e *TemplateEngine) Context() *TemplateContext {
	return e.ctx
}

// RenderTemplate renders a template string using the engine's context.
func (e *TemplateEngine) RenderTemplate(name, content string) (string, error) {
	// Create template with Sprig functions
	tmpl, err := template.New(name).
		Funcs(sprig.TxtFuncMap()).
		Parse(content)
	if err != nil {
		return "", fmt.Errorf("failed to parse template %s: %w", name, err)
	}

	// Execute template with context
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, e.ctx); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", name, err)
	}

	return buf.String(), nil
}

// templateContextWithVars extends TemplateContext with manifest-level variables
// accessible as .Vars.* in templates. Used only during per-file rendering.
type templateContextWithVars struct {
	Values map[string]any
	Env    map[string]string
	OS     OSInfo
	Vars   map[string]any
}

// RenderTemplateWithVars renders a template with the engine's context extended by
// manifest-level vars accessible as .Vars.*. Use this when a ManagedFile spec
// defines vars: to override or augment the global template context.
func (e *TemplateEngine) RenderTemplateWithVars(name, content string, vars map[string]any) (string, error) {
	ctx := templateContextWithVars{
		Values: e.ctx.Values,
		Env:    e.ctx.Env,
		OS:     e.ctx.OS,
		Vars:   vars,
	}

	tmpl, err := template.New(name).
		Funcs(sprig.TxtFuncMap()).
		Parse(content)
	if err != nil {
		return "", fmt.Errorf("failed to parse template %s: %w", name, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", name, err)
	}

	return buf.String(), nil
}

// LoadAndRenderValues loads values.yaml from the given path, renders it as a template
// using the current context (Env and OS), and parses the result into ctx.Values.
// This is the first-pass templating step.
func (e *TemplateEngine) LoadAndRenderValues(valuesPath string) error {
	// Check if file exists
	if _, err := os.Stat(valuesPath); os.IsNotExist(err) {
		slog.Warn("values.yaml not found — .Values will be empty in templates", "path", valuesPath)
		return nil
	}

	// Read raw values.yaml
	data, err := os.ReadFile(valuesPath)
	if err != nil {
		return fmt.Errorf("failed to read values file %s: %w", valuesPath, err)
	}

	// Render as template
	rendered, err := e.RenderTemplate("values.yaml", string(data))
	if err != nil {
		return fmt.Errorf("failed to render values template: %w", err)
	}

	// Parse rendered YAML into Values
	if err := yaml.Unmarshal([]byte(rendered), &e.ctx.Values); err != nil {
		return fmt.Errorf("failed to parse rendered values.yaml: %w", err)
	}

	return nil
}

// RenderString is a convenience method to render a template string.
// It creates a temporary engine with the given context.
func RenderString(ctx *TemplateContext, content string) (string, error) {
	engine := NewTemplateEngine(ctx)
	return engine.RenderTemplate("string", content)
}
