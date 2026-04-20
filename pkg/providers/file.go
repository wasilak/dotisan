// Package providers implements resource providers for dotisan.
//
// Providers are the core abstraction that enables dotisan to manage different
// types of resources:
//   - FileProvider: Manages files and directories (ManagedFile, ManagedDirectory)
//   - BrewProvider: Manages Homebrew packages
//   - NpmProvider: Manages npm packages
//   - GoProvider: Manages Go modules
//   - CargoProvider: Manages Rust crates
//
// Each provider implements the provider.Provider interface.
package providers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dotisan/pkg/config"
	"dotisan/pkg/diff"
	"dotisan/pkg/provider"
	"dotisan/pkg/resource"
)

// FileProvider implements the Provider interface for ManagedFile and ManagedDirectory resources.
type FileProvider struct {
	// templateContext provides access to template variables
	templateContext *config.TemplateContext

	// diffEngine provides diff generation capabilities
	diffEngine *diff.Engine

	// dotfilesRoot is the path to the dotfiles directory
	dotfilesRoot string
}

// NewFileProvider creates a new FileProvider.
func NewFileProvider(ctx *config.TemplateContext, engine *diff.Engine, dotfilesRoot string) *FileProvider {
	return &FileProvider{
		templateContext: ctx,
		diffEngine:      engine,
		dotfilesRoot:    dotfilesRoot,
	}
}

// Name returns the provider name.
func (p *FileProvider) Name() string {
	return "file"
}

// Available checks if the provider can operate on this system.
// File operations are always available if we have filesystem access.
func (p *FileProvider) Available() (bool, string) {
	// Check if we can read the dotfiles root
	if p.dotfilesRoot != "" {
		if _, err := os.Stat(p.dotfilesRoot); err != nil {
			return false, fmt.Sprintf("dotfiles root not accessible: %s", err)
		}
	}

	// Check if we have write access to home directory for state file
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false, fmt.Sprintf("cannot determine home directory: %s", err)
	}

	// Try to stat home directory
	if _, err := os.Stat(homeDir); err != nil {
		return false, fmt.Sprintf("home directory not accessible: %s", err)
	}

	return true, "filesystem operations available"
}

// Reconcile compares the desired resources with the current system state
// and returns a plan of changes needed.
func (p *FileProvider) Reconcile(desired []resource.Resource, currentState []provider.ResourceState) provider.Plan {
	plan := provider.Plan{}

	// Build a map of current state for quick lookup
	stateMap := make(map[string]provider.ResourceState)
	for _, s := range currentState {
		stateMap[s.ID] = s
	}

	// Track which resources are in the desired state
	desiredIDs := make(map[string]bool)

	for _, res := range desired {
		switch r := res.(type) {
		case *resource.ManagedFile:
			p.reconcileManagedFile(r, stateMap, &plan, desiredIDs)
		case *resource.ManagedDirectory:
			// TODO: Implement in subtask 7.4
		}
	}

	// Check for resources that should be removed (in state but not in desired)
	for id, s := range stateMap {
		if !desiredIDs[id] && (s.Kind == "ManagedFile" || s.Kind == "ManagedDirectory") {
			// Find the resource in desired to get full metadata
			plan.Removals = append(plan.Removals, &resource.ManagedFile{
				BaseResource: resource.BaseResource{
					Kind: s.Kind,
					Metadata: resource.Metadata{
						Name:      s.Name,
						Namespace: s.Namespace,
					},
				},
			})
		}
	}

	return plan
}

// reconcileManagedFile reconciles a single ManagedFile resource.
func (p *FileProvider) reconcileManagedFile(
	mf *resource.ManagedFile,
	stateMap map[string]provider.ResourceState,
	plan *provider.Plan,
	desiredIDs map[string]bool,
) {
	// Build resource ID
	id := fmt.Sprintf("file/%s/%s", mf.GetMetadata().GetNamespace(), mf.GetMetadata().Name)
	desiredIDs[id] = true

	// Resolve source and destination paths
	sourcePath := filepath.Join(p.dotfilesRoot, mf.Spec.Source)
	destPath, err := p.resolveDestination(mf.Spec.Destination)
	if err != nil {
		// Can't resolve destination, mark as error
		return
	}

	// Render source content
	content, err := p.renderSource(sourcePath, mf.Spec.Template)
	if err != nil {
		// Can't read/render source, mark as error
		return
	}

	// Calculate checksum
	checksum := calculateChecksum(content)

	// Get current state
	savedState, hasSavedState := stateMap[id]

	// Check if file exists at destination
	_, err = os.Stat(destPath)
	fileExists := err == nil

	// Build desired state
	desiredState := provider.ResourceState{
		ID:         id,
		Kind:       "ManagedFile",
		Name:       mf.GetMetadata().Name,
		Namespace:  mf.GetMetadata().GetNamespace(),
		Checksum:   checksum,
		DestHash:   checksum, // For files, dest_hash is the rendered content hash
		SourceHash: calculateChecksumFromFile(sourcePath), // Hash of source file
		Extra: map[string]interface{}{
			"source_path": sourcePath,
			"dest_path":   destPath,
			"mode":        mf.Spec.Mode,
		},
	}

	// Determine action
	if !fileExists {
		// File doesn't exist - needs to be created
		plan.Additions = append(plan.Additions, mf)
		return
	}

	if !hasSavedState {
		// File exists but wasn't managed by us - check if it matches desired
		actualContent, err := os.ReadFile(destPath)
		if err != nil {
			return
		}
		actualChecksum := calculateChecksum(string(actualContent))

		if actualChecksum != checksum {
			// File exists with different content - drift
			plan.Modifications = append(plan.Modifications, provider.Modification{
				Resource:  mf,
				OldState:  provider.ResourceState{ID: id, DestHash: actualChecksum},
				NewState:  desiredState,
				Diff:      p.generateDiff(string(actualContent), content),
			})
		} else {
			// File exists with same content - in sync
			plan.InSync = append(plan.InSync, mf)
		}
		return
	}

	// File exists and we have saved state
	actualContent, err := os.ReadFile(destPath)
	if err != nil {
		return
	}
	actualChecksum := calculateChecksum(string(actualContent))

	// Check if file has drifted (changed outside of dotisan)
	if actualChecksum != savedState.DestHash {
		// File has been modified outside of dotisan
		plan.Drifted = append(plan.Drifted, provider.Drift{
			Resource:      mf,
			ExpectedState: savedState,
			ActualState:   provider.ResourceState{ID: id, DestHash: actualChecksum},
			Description:   "file content has changed",
		})
		return
	}

	// Check if desired state has changed
	if checksum != savedState.DestHash {
		plan.Modifications = append(plan.Modifications, provider.Modification{
			Resource: mf,
			OldState: savedState,
			NewState: desiredState,
			Diff:     p.generateDiff(string(actualContent), content),
		})
		return
	}

	// File is in sync
	plan.InSync = append(plan.InSync, mf)
}

// resolveDestination resolves the destination path using template variables.
func (p *FileProvider) resolveDestination(dest string) (string, error) {
	if p.templateContext == nil {
		return dest, nil
	}

	// Use the template engine to resolve the destination
	engine := config.NewTemplateEngine(p.templateContext)
	resolved, err := engine.RenderTemplate("destination", dest)
	if err != nil {
		return "", fmt.Errorf("failed to resolve destination: %w", err)
	}

	// Expand ~ to home directory
	if strings.HasPrefix(resolved, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		resolved = filepath.Join(homeDir, resolved[2:])
	}

	return resolved, nil
}

// renderSource renders the source file (if template is enabled) or reads it as-is.
func (p *FileProvider) renderSource(sourcePath string, isTemplate bool) (string, error) {
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", fmt.Errorf("failed to read source file %s: %w", sourcePath, err)
	}

	if !isTemplate || p.templateContext == nil {
		return string(content), nil
	}

	// Render as template
	engine := config.NewTemplateEngine(p.templateContext)
	rendered, err := engine.RenderTemplate(sourcePath, string(content))
	if err != nil {
		return "", fmt.Errorf("failed to render template %s: %w", sourcePath, err)
	}

	return rendered, nil
}

// calculateChecksum calculates SHA256 checksum of content.
func calculateChecksum(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// calculateChecksumFromFile calculates SHA256 checksum of a file.
func calculateChecksumFromFile(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return calculateChecksum(string(content))
}

// generateDiff generates a diff between old and new content.
func (p *FileProvider) generateDiff(oldContent, newContent string) string {
	if p.diffEngine == nil {
		return ""
	}
	// Generate a short diff preview (first few lines)
	changes := p.diffEngine.GenerateDiff(oldContent, newContent)
	if len(changes) == 0 {
		return ""
	}

	var result strings.Builder
	for i, change := range changes {
		if i > 10 {
			result.WriteString("...")
			break
		}
		if change.Type != diff.LineUnchanged {
			result.WriteString(change.Type.String())
			result.WriteString(" ")
			result.WriteString(change.Content)
			result.WriteString("\n")
		}
	}

	return result.String()
}

// Apply executes the given plan.
func (p *FileProvider) Apply(ctx context.Context, plan provider.Plan) error {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return err
	}

	// Process additions
	for _, res := range plan.Additions {
		if err := p.applyAddition(ctx, res); err != nil {
			return fmt.Errorf("failed to add %s: %w", res.GetMetadata().ResourceID(), err)
		}
	}

	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return err
	}

	// Process modifications
	for _, mod := range plan.Modifications {
		if err := p.applyModification(ctx, mod); err != nil {
			return fmt.Errorf("failed to modify %s: %w", mod.Resource.GetMetadata().ResourceID(), err)
		}
	}

	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return err
	}

	// Process removals
	for _, res := range plan.Removals {
		if err := p.applyRemoval(ctx, res); err != nil {
			return fmt.Errorf("failed to remove %s: %w", res.GetMetadata().ResourceID(), err)
		}
	}

	return nil
}

// applyAddition creates a new file.
func (p *FileProvider) applyAddition(ctx context.Context, res resource.Resource) error {
	mf, ok := res.(*resource.ManagedFile)
	if !ok {
		return fmt.Errorf("not a ManagedFile resource")
	}

	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return err
	}

	// Resolve paths
	sourcePath := filepath.Join(p.dotfilesRoot, mf.Spec.Source)
	destPath, err := p.resolveDestination(mf.Spec.Destination)
	if err != nil {
		return err
	}

	// Render content
	content, err := p.renderSource(sourcePath, mf.Spec.Template)
	if err != nil {
		return err
	}

	// Ensure parent directory exists
	parentDir := filepath.Dir(destPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory %s: %w", parentDir, err)
	}

	// Write file
	if err := os.WriteFile(destPath, []byte(content), p.parseMode(mf.Spec.Mode)); err != nil {
		return fmt.Errorf("failed to write file %s: %w", destPath, err)
	}

	return nil
}

// applyModification updates an existing file.
func (p *FileProvider) applyModification(ctx context.Context, mod provider.Modification) error {
	mf, ok := mod.Resource.(*resource.ManagedFile)
	if !ok {
		return fmt.Errorf("not a ManagedFile resource")
	}

	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return err
	}

	// Resolve paths
	sourcePath := filepath.Join(p.dotfilesRoot, mf.Spec.Source)
	destPath, err := p.resolveDestination(mf.Spec.Destination)
	if err != nil {
		return err
	}

	// Render content
	content, err := p.renderSource(sourcePath, mf.Spec.Template)
	if err != nil {
		return err
	}

	// Write file (this overwrites existing content)
	if err := os.WriteFile(destPath, []byte(content), p.parseMode(mf.Spec.Mode)); err != nil {
		return fmt.Errorf("failed to write file %s: %w", destPath, err)
	}

	return nil
}

// applyRemoval deletes a file.
func (p *FileProvider) applyRemoval(ctx context.Context, res resource.Resource) error {
	mf, ok := res.(*resource.ManagedFile)
	if !ok {
		return fmt.Errorf("not a ManagedFile resource")
	}

	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return err
	}

	// Resolve destination path
	destPath, err := p.resolveDestination(mf.Spec.Destination)
	if err != nil {
		return err
	}

	// Check if file exists
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		// File doesn't exist, nothing to do
		return nil
	}

	// Delete the file
	if err := os.Remove(destPath); err != nil {
		return fmt.Errorf("failed to remove file %s: %w", destPath, err)
	}

	return nil
}

// parseMode parses a file mode string (e.g., "0644") into os.FileMode.
// Returns 0644 (rw-r--r--) if mode is empty or invalid.
func (p *FileProvider) parseMode(mode string) os.FileMode {
	if mode == "" {
		return 0644
	}

	// Parse octal mode
	var m uint32
	if _, err := fmt.Sscanf(mode, "%o", &m); err != nil {
		return 0644
	}

	return os.FileMode(m)
}

// Import discovers an existing resource on the system and returns its state.
func (p *FileProvider) Import(ctx context.Context, id string) (provider.ResourceState, error) {
	// TODO: Implement import functionality
	return provider.ResourceState{}, fmt.Errorf("import not yet implemented")
}
