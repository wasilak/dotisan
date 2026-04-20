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

	"github.com/wasilak/dotisan/pkg/config"
	"github.com/wasilak/dotisan/pkg/diff"
	"github.com/wasilak/dotisan/pkg/provider"
	"github.com/wasilak/dotisan/pkg/resource"
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
			p.reconcileManagedDirectory(r, stateMap, &plan, desiredIDs)
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
	// Build resource ID using kind and name (Terraform-style: files/dirs are for human org only)
	id := fmt.Sprintf("ManagedFile/%s", mf.GetMetadata().Name)
	desiredIDs[id] = true

	// Resolve destination path
	destPath, err := p.resolveDestination(mf.Spec.Destination)
	if err != nil {
		// Can't resolve destination, mark as error
		return
	}

	// Render source content (either from inline Source or external SourceFile)
	var content string
	var sourcePath string

	if mf.Spec.Source != "" {
		// Inline content
		sourcePath = "" // No source file for inline content
		if mf.Spec.Template && p.templateContext != nil {
			// Apply templating to inline content
			engine := config.NewTemplateEngine(p.templateContext)
			var renderErr error
			content, renderErr = engine.RenderTemplate("inline", mf.Spec.Source)
			if renderErr != nil {
				// Template rendering failed
				plan.Additions = append(plan.Additions, mf)
				return
			}
		} else {
			// Use inline content directly without templating
			content = mf.Spec.Source
		}
	} else if mf.Spec.SourceFile != "" {
		// External file path (relative to resources/ directory)
		sourcePath = filepath.Join(p.dotfilesRoot, "resources", mf.Spec.SourceFile)
		var renderErr error
		content, renderErr = p.renderSource(sourcePath, mf.Spec.Template)
		if renderErr != nil {
			// Can't read/render source, mark as error - but still add to plan so user knows
			plan.Additions = append(plan.Additions, mf)
			return
		}
	} else {
		// Neither Source nor SourceFile set - validation should catch this
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
		DestHash:   checksum,                              // For files, dest_hash is the rendered content hash
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
		// File exists but wasn't managed by us - treat as addition and warn the user
		// This avoids silently marking unmanaged files as in-sync and gives a clear
		// suggestion to import the resource into state.
		if fileExists {
			plan.Additions = append(plan.Additions, mf)

			// Build a copy-pasteable suggestion for importing this resource
			suggestion := fmt.Sprintf("dotisan state import ManagedFile %s %s", mf.GetMetadata().Name, destPath)
			warning := provider.PlanWarning{
				ResourceID: id,
				Severity:   "warning",
				Message:    fmt.Sprintf("Destination file already exists at %s", destPath),
				Suggestion: suggestion,
			}
			plan.Warnings = append(plan.Warnings, warning)
		}
		return
	}

	// File exists and we have saved state
	actualContent, err := os.ReadFile(destPath)
	if err != nil {
		return
	}
	actualChecksum := calculateChecksum(string(actualContent))

	// Special handling for imported resources
	// When a resource was imported, we compare the actual destination file
	// with the saved state (which contains the hash of the file at import time).
	// If they match, the resource is in sync. We then clear the imported flag.
	if imported, ok := savedState.Extra["imported"].(bool); ok && imported {
		if actualChecksum == savedState.DestHash {
			// Imported file still matches what we saved - it's in sync
			// Clear the imported flag for future normal reconciliation
			plan.InSync = append(plan.InSync, mf)
			// Note: The imported flag will be cleared when state is saved after apply
			// because the resourceToStateEntry function doesn't preserve the imported flag
			return
		}
		// File was modified after import - this is drift
		plan.Drifted = append(plan.Drifted, provider.Drift{
			Resource:      mf,
			ExpectedState: savedState,
			ActualState:   provider.ResourceState{ID: id, DestHash: actualChecksum},
			Description:   "file content has changed since import",
			Diff:          p.generateDiff(string(actualContent), content),
		})
		return
	}

	// Check if file has drifted (changed outside of dotisan)
	if actualChecksum != savedState.DestHash {
		// File has been modified outside of dotisan
		// The expected content is our rendered content (what the file SHOULD be)
		// actualContent is what the file currently IS

		plan.Drifted = append(plan.Drifted, provider.Drift{
			Resource:      mf,
			ExpectedState: savedState,
			ActualState:   provider.ResourceState{ID: id, DestHash: actualChecksum},
			Description:   "file content has changed",
			Diff:          p.generateDiff(content, string(actualContent)),
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

// reconcileManagedDirectory reconciles a single ManagedDirectory resource.
func (p *FileProvider) reconcileManagedDirectory(
	md *resource.ManagedDirectory,
	stateMap map[string]provider.ResourceState,
	plan *provider.Plan,
	desiredIDs map[string]bool,
) {
	// Build resource ID using kind and name (Terraform-style: files/dirs are for human org only)
	id := fmt.Sprintf("ManagedDirectory/%s", md.GetMetadata().Name)
	desiredIDs[id] = true

	// Resolve source and destination paths (source is relative to resources/ directory)
	sourcePath := filepath.Join(p.dotfilesRoot, "resources", md.Spec.SourceDir)
	destPath, err := p.resolveDestination(md.Spec.Destination)
	if err != nil {
		return
	}

	// Check if source directory exists
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return
	}
	if !sourceInfo.IsDir() {
		return
	}

	// Check if destination directory exists
	destExists := false
	destInfo, err := os.Stat(destPath)
	if err == nil && destInfo.IsDir() {
		destExists = true
	}

	// Get current state
	savedState, hasSavedState := stateMap[id]

	// Build list of all files in source (for comparison)
	sourceFiles := make(map[string]string) // relative path -> checksum
	if err := p.walkSourceDir(sourcePath, "", md.Spec.Recursive, md.Spec.Exclude, sourceFiles); err != nil {
		return
	}

	// Build list of all files in destination (for clean operation)
	destFiles := make(map[string]string) // relative path -> checksum
	if destExists {
		if err := p.walkDestDir(destPath, "", md.Spec.Recursive, destFiles); err != nil {
			return
		}
	}

	// Determine changes
	var hasChanges bool

	// Check for new and modified files
	for relPath, sourceChecksum := range sourceFiles {
		destChecksum, exists := destFiles[relPath]

		if !exists {
			// New file to add
			hasChanges = true
		} else if sourceChecksum != destChecksum {
			// File modified
			hasChanges = true
		}
	}

	// Check for files to remove (clean operation)
	if md.Spec.Clean {
		for relPath := range destFiles {
			if _, exists := sourceFiles[relPath]; !exists {
				// File exists in dest but not in source - should be removed
				hasChanges = true
			}
		}
	}

	// Build desired state
	desiredState := provider.ResourceState{
		ID:        id,
		Kind:      "ManagedDirectory",
		Name:      md.GetMetadata().Name,
		Namespace: md.GetMetadata().GetNamespace(),
		Extra: map[string]interface{}{
			"source_path": sourcePath,
			"dest_path":   destPath,
			"recursive":   md.Spec.Recursive,
			"clean":       md.Spec.Clean,
			"exclude":     md.Spec.Exclude,
		},
	}

	// Determine action
	if !destExists {
		// Directory doesn't exist - needs to be created (with all contents)
		plan.Additions = append(plan.Additions, md)
		return
	}

	if !hasSavedState {
		// Directory exists but wasn't managed by us
		if hasChanges {
			plan.Modifications = append(plan.Modifications, provider.Modification{
				Resource: md,
				OldState: provider.ResourceState{ID: id},
				NewState: desiredState,
				Diff:     fmt.Sprintf("directory %s has changes", destPath),
			})
		} else {
			plan.InSync = append(plan.InSync, md)
		}
		return
	}

	// Directory exists and we have saved state
	if hasChanges {
		plan.Modifications = append(plan.Modifications, provider.Modification{
			Resource: md,
			OldState: savedState,
			NewState: desiredState,
			Diff:     fmt.Sprintf("directory %s has changes", destPath),
		})
		return
	}

	// Directory is in sync
	plan.InSync = append(plan.InSync, md)
}

// walkSourceDir recursively walks the source directory and builds a map of files.
func (p *FileProvider) walkSourceDir(root, rel string, recursive bool, exclude []string, files map[string]string) error {
	fullPath := filepath.Join(root, rel)

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		entryRel := filepath.Join(rel, entry.Name())

		// Check if this path should be excluded
		if p.shouldExclude(entryRel, exclude) {
			continue
		}

		if entry.IsDir() {
			if recursive {
				if err := p.walkSourceDir(root, entryRel, recursive, exclude, files); err != nil {
					return err
				}
			}
		} else {
			// Calculate checksum for this file
			filePath := filepath.Join(root, entryRel)
			checksum := calculateChecksumFromFile(filePath)
			if checksum != "" {
				files[entryRel] = checksum
			}
		}
	}

	return nil
}

// walkDestDir recursively walks the destination directory and builds a map of files.
func (p *FileProvider) walkDestDir(root, rel string, recursive bool, files map[string]string) error {
	fullPath := filepath.Join(root, rel)

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		entryRel := filepath.Join(rel, entry.Name())

		if entry.IsDir() {
			if recursive {
				if err := p.walkDestDir(root, entryRel, recursive, files); err != nil {
					return err
				}
			}
		} else {
			// Calculate checksum for this file
			filePath := filepath.Join(root, entryRel)
			checksum := calculateChecksumFromFile(filePath)
			if checksum != "" {
				files[entryRel] = checksum
			}
		}
	}

	return nil
}

// shouldExclude checks if a path matches any of the exclude patterns.
func (p *FileProvider) shouldExclude(path string, exclude []string) bool {
	for _, pattern := range exclude {
		matched, err := filepath.Match(pattern, path)
		if err == nil && matched {
			return true
		}
		// Also check just the filename
		matched, err = filepath.Match(pattern, filepath.Base(path))
		if err == nil && matched {
			return true
		}
	}
	return false
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
	oldLines := strings.Split(strings.TrimSpace(oldContent), "\n")
	newLines := strings.Split(strings.TrimSpace(newContent), "\n")

	var result []string
	maxLines := 20 // Limit diff output

	// Find removed lines (in old but not in new)
	for _, oldLine := range oldLines {
		if oldLine == "" {
			continue
		}
		found := false
		for _, newLine := range newLines {
			if oldLine == newLine {
				found = true
				break
			}
		}
		if !found {
			result = append(result, "- "+oldLine)
			if len(result) >= maxLines {
				result = append(result, "...")
				break
			}
		}
	}

	if len(result) >= maxLines {
		return strings.Join(result, "\n")
	}

	// Find added lines (in new but not in old)
	for _, newLine := range newLines {
		if newLine == "" {
			continue
		}
		found := false
		for _, oldLine := range oldLines {
			if oldLine == newLine {
				found = true
				break
			}
		}
		if !found {
			result = append(result, "+ "+newLine)
			if len(result) >= maxLines {
				result = append(result, "...")
				break
			}
		}
	}

	return strings.Join(result, "\n")
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

// applyAddition creates a new resource (file or directory).
func (p *FileProvider) applyAddition(ctx context.Context, res resource.Resource) error {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return err
	}

	switch r := res.(type) {
	case *resource.ManagedFile:
		return p.applyFileAddition(ctx, r)
	case *resource.ManagedDirectory:
		return p.applyDirectoryAddition(ctx, r)
	default:
		return fmt.Errorf("unsupported resource type for addition: %T", res)
	}
}

// applyFileAddition creates a new file.
func (p *FileProvider) applyFileAddition(ctx context.Context, mf *resource.ManagedFile) error {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return err
	}

	// Resolve destination path
	destPath, err := p.resolveDestination(mf.Spec.Destination)
	if err != nil {
		return err
	}

	// Render content (from inline Source or external SourceFile)
	var content string

	if mf.Spec.Source != "" {
		// Inline content
		if mf.Spec.Template && p.templateContext != nil {
			// Apply templating to inline content
			engine := config.NewTemplateEngine(p.templateContext)
			content, err = engine.RenderTemplate("inline", mf.Spec.Source)
			if err != nil {
				return fmt.Errorf("failed to render template: %w", err)
			}
		} else {
			// Use inline content directly
			content = mf.Spec.Source
		}
	} else if mf.Spec.SourceFile != "" {
		// External file path (relative to resources/ directory)
		sourcePath := filepath.Join(p.dotfilesRoot, "resources", mf.Spec.SourceFile)
		content, err = p.renderSource(sourcePath, mf.Spec.Template)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("neither source nor sourceFile specified")
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

// applyDirectoryAddition creates a new directory by copying from source.
func (p *FileProvider) applyDirectoryAddition(ctx context.Context, md *resource.ManagedDirectory) error {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return err
	}

	// Resolve paths (source is relative to resources/ directory)
	sourcePath := filepath.Join(p.dotfilesRoot, "resources", md.Spec.SourceDir)
	destPath, err := p.resolveDestination(md.Spec.Destination)
	if err != nil {
		return err
	}

	// Ensure destination directory exists
	if err := os.MkdirAll(destPath, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory %s: %w", destPath, err)
	}

	// Copy all files from source to destination
	return p.syncDirectory(ctx, sourcePath, destPath, md.Spec.Recursive, md.Spec.Exclude, md.Spec.Clean)
}

// applyModification updates an existing resource.
func (p *FileProvider) applyModification(ctx context.Context, mod provider.Modification) error {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return err
	}

	switch r := mod.Resource.(type) {
	case *resource.ManagedFile:
		return p.applyFileModification(ctx, r)
	case *resource.ManagedDirectory:
		return p.applyDirectoryModification(ctx, r)
	default:
		return fmt.Errorf("unsupported resource type for modification: %T", mod.Resource)
	}
}

// applyFileModification updates an existing file.
func (p *FileProvider) applyFileModification(ctx context.Context, mf *resource.ManagedFile) error {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return err
	}

	// Resolve destination path
	destPath, err := p.resolveDestination(mf.Spec.Destination)
	if err != nil {
		return err
	}

	// Render content (from inline Source or external SourceFile)
	var content string

	if mf.Spec.Source != "" {
		// Inline content
		if mf.Spec.Template && p.templateContext != nil {
			// Apply templating to inline content
			engine := config.NewTemplateEngine(p.templateContext)
			content, err = engine.RenderTemplate("inline", mf.Spec.Source)
			if err != nil {
				return fmt.Errorf("failed to render template: %w", err)
			}
		} else {
			// Use inline content directly
			content = mf.Spec.Source
		}
	} else if mf.Spec.SourceFile != "" {
		// External file path (relative to resources/ directory)
		sourcePath := filepath.Join(p.dotfilesRoot, "resources", mf.Spec.SourceFile)
		content, err = p.renderSource(sourcePath, mf.Spec.Template)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("neither source nor sourceFile specified")
	}

	// Write file (this overwrites existing content)
	if err := os.WriteFile(destPath, []byte(content), p.parseMode(mf.Spec.Mode)); err != nil {
		return fmt.Errorf("failed to write file %s: %w", destPath, err)
	}

	return nil
}

// applyDirectoryModification updates an existing directory.
func (p *FileProvider) applyDirectoryModification(ctx context.Context, md *resource.ManagedDirectory) error {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return err
	}

	// Resolve paths (source is relative to resources/ directory)
	sourcePath := filepath.Join(p.dotfilesRoot, "resources", md.Spec.SourceDir)
	destPath, err := p.resolveDestination(md.Spec.Destination)
	if err != nil {
		return err
	}

	// Sync directory contents
	return p.syncDirectory(ctx, sourcePath, destPath, md.Spec.Recursive, md.Spec.Exclude, md.Spec.Clean)
}

// applyRemoval deletes a resource.
func (p *FileProvider) applyRemoval(ctx context.Context, res resource.Resource) error {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return err
	}

	switch r := res.(type) {
	case *resource.ManagedFile:
		return p.applyFileRemoval(ctx, r)
	case *resource.ManagedDirectory:
		return p.applyDirectoryRemoval(ctx, r)
	default:
		return fmt.Errorf("unsupported resource type for removal: %T", res)
	}
}

// applyFileRemoval deletes a file.
func (p *FileProvider) applyFileRemoval(ctx context.Context, mf *resource.ManagedFile) error {
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

// applyDirectoryRemoval deletes a directory.
func (p *FileProvider) applyDirectoryRemoval(ctx context.Context, md *resource.ManagedDirectory) error {
	// Resolve destination path
	destPath, err := p.resolveDestination(md.Spec.Destination)
	if err != nil {
		return err
	}

	// Check if directory exists
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		// Directory doesn't exist, nothing to do
		return nil
	}

	// Delete the directory and all contents
	if err := os.RemoveAll(destPath); err != nil {
		return fmt.Errorf("failed to remove directory %s: %w", destPath, err)
	}

	return nil
}

// syncDirectory synchronizes files from source to destination directory.
func (p *FileProvider) syncDirectory(ctx context.Context, sourcePath, destPath string, recursive bool, exclude []string, clean bool) error {
	// Build list of source files
	sourceFiles := make(map[string]string)
	if err := p.walkSourceDir(sourcePath, "", recursive, exclude, sourceFiles); err != nil {
		return fmt.Errorf("failed to walk source directory: %w", err)
	}

	// Build list of destination files
	destFiles := make(map[string]string)
	if err := p.walkDestDir(destPath, "", recursive, destFiles); err != nil {
		// Destination might not exist yet, that's ok
	}

	// Copy new and modified files
	for relPath, sourceChecksum := range sourceFiles {
		// Check context cancellation
		if err := ctx.Err(); err != nil {
			return err
		}

		destFilePath := filepath.Join(destPath, relPath)
		destChecksum, exists := destFiles[relPath]

		if !exists || sourceChecksum != destChecksum {
			// File needs to be copied
			sourceFilePath := filepath.Join(sourcePath, relPath)
			content, err := os.ReadFile(sourceFilePath)
			if err != nil {
				return fmt.Errorf("failed to read source file %s: %w", sourceFilePath, err)
			}

			// Ensure parent directory exists
			parentDir := filepath.Dir(destFilePath)
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				return fmt.Errorf("failed to create parent directory %s: %w", parentDir, err)
			}

			// Write file
			if err := os.WriteFile(destFilePath, content, 0644); err != nil {
				return fmt.Errorf("failed to write file %s: %w", destFilePath, err)
			}
		}
	}

	// Clean up extra files if requested
	if clean {
		for relPath := range destFiles {
			// Check context cancellation
			if err := ctx.Err(); err != nil {
				return err
			}

			if _, exists := sourceFiles[relPath]; !exists {
				// File exists in dest but not in source - remove it
				destFilePath := filepath.Join(destPath, relPath)
				if err := os.Remove(destFilePath); err != nil {
					return fmt.Errorf("failed to remove file %s: %w", destFilePath, err)
				}
			}
		}
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
	// id is the file path (e.g., "~/.zshrc")
	// Expand home directory if needed
	filePath := id
	if strings.HasPrefix(filePath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return provider.ResourceState{}, fmt.Errorf("failed to get home directory: %w", err)
		}
		filePath = filepath.Join(homeDir, filePath[2:])
	}

	// Check if file exists
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return provider.ResourceState{}, fmt.Errorf("file %s does not exist", id)
		}
		return provider.ResourceState{}, fmt.Errorf("failed to stat file %s: %w", id, err)
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return provider.ResourceState{}, fmt.Errorf("failed to read file %s: %w", id, err)
	}

	// Calculate checksum
	checksum := calculateChecksum(string(content))

	// Get file mode
	mode := fmt.Sprintf("%04o", fileInfo.Mode().Perm())

	// Get the base name and clean it up (remove leading dot if present)
	baseName := filepath.Base(id)
	if strings.HasPrefix(baseName, ".") {
		baseName = baseName[1:] // Remove leading dot for cleaner resource names
	}

	return provider.ResourceState{
		ID:       fmt.Sprintf("ManagedFile/%s", baseName),
		Kind:     "ManagedFile",
		Name:     baseName,
		DestHash: checksum,
		Extra: map[string]interface{}{
			"source_path": "",
			"dest_path":   filePath,
			"mode":        mode,
			"imported":    true,
		},
	}, nil
}
