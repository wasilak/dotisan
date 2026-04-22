package resource

import "fmt"

// ManagedDirectory defines a directory to manage recursively.
// Can optionally clean (remove files not in source).
type ManagedDirectory struct {
	BaseResource `yaml:",inline"`
	Spec         ManagedDirectorySpec `yaml:"spec" validate:"required"`
}

// ManagedDirectorySpec contains the ManagedDirectory configuration.
// Exactly one of Source (inline file list) or SourceDir (external directory) must be specified.
type ManagedDirectorySpec struct {
	// Source is inline file content map (rarely used, for special cases)
	// Normally you should use SourceDir for directory sync
	Source string `yaml:"source,omitempty"`

	// SourceDir is a path to the source directory (relative to dotfiles root)
	// Example: sourceDir: configs/nvim
	SourceDir string `yaml:"sourceDir,omitempty"`

	// Destination is the absolute path where the directory should be placed
	// Can contain template expressions
	Destination string `yaml:"destination" validate:"required"`

	// Recursive indicates if subdirectories should be synced
	Recursive bool `yaml:"recursive,omitempty"`

	// Clean indicates if files at destination not in source should be removed
	Clean bool `yaml:"clean,omitempty"`

	// Exclude is a list of glob patterns to exclude from sync
	Exclude []string `yaml:"exclude,omitempty"`

	// Directories is a list of directories to manage (new list-based syntax).
	// When populated, this takes precedence over the single-directory fields above.
	Directories []DirectorySpec `yaml:"directories,omitempty"`
}

// DirectorySpec represents a single directory in the list-based ManagedDirectory structure.
type DirectorySpec struct {
	SourceDir   string   `yaml:"sourceDir,omitempty"`
	Destination string   `yaml:"destination" validate:"required"`
	Recursive   bool     `yaml:"recursive,omitempty"`
	Clean       bool     `yaml:"clean,omitempty"`
	Exclude     []string `yaml:"exclude,omitempty"`
}

// Validate implements Resource.Validate.
func (r ManagedDirectory) Validate() error {
	// Standard struct validation
	if err := ValidateStruct(r); err != nil {
		return err
	}

	// Custom validation for single-directory syntax (backward compatibility)
	if len(r.Spec.Directories) == 0 {
		// Single-directory mode: at most one of Source or SourceDir should be set
		hasSource := r.Spec.Source != ""
		hasSourceDir := r.Spec.SourceDir != ""

		if hasSource && hasSourceDir {
			return fmt.Errorf("ManagedDirectory.spec: 'source' and 'sourceDir' are mutually exclusive, use exactly one")
		}
	}

	return nil
}

// GetDirectories returns the list of directories to manage.
// If Directories is populated (new syntax), returns that.
// Otherwise, converts the single-directory fields to a list (backward compatibility).
func (r ManagedDirectory) GetDirectories() []DirectorySpec {
	if len(r.Spec.Directories) > 0 {
		return r.Spec.Directories
	}

	// Convert single-directory to list for backward compatibility
	return []DirectorySpec{{
		SourceDir:   r.Spec.SourceDir,
		Destination: r.Spec.Destination,
		Recursive:   r.Spec.Recursive,
		Clean:       r.Spec.Clean,
		Exclude:     r.Spec.Exclude,
	}}
}
