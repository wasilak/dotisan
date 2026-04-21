package resource

import "fmt"

// ManagedFile defines a single file to manage.
// The file can be templated or static, with content from inline source or external file.
type ManagedFile struct {
	BaseResource `yaml:",inline"`
	Spec         ManagedFileSpec `yaml:"spec" validate:"required"`
}

// ManagedFileSpec contains the ManagedFile configuration.
// Exactly one of Source (inline) or SourceFile (external file) must be specified.
type ManagedFileSpec struct {
	// Source is inline content (use | in YAML for multi-line)
	// Example:
	//   source: |
	//     # My config
	//     export EDITOR=vim
	Source string `yaml:"source,omitempty"`

	// SourceFile is a path to an external file (relative to dotfiles root)
	// Example: sourceFile: shell/zshrc.sh
	SourceFile string `yaml:"sourceFile,omitempty"`

	// Destination is the absolute path where the file should be placed
	// Can contain template expressions like "{{ .Env.HOME }}/.zshrc"
	Destination string `yaml:"destination" validate:"required"`

	// Template indicates if the source should be processed as a Go template
	Template bool `yaml:"template,omitempty"`

	// Mode is the file permissions (e.g., "0644", "0755")
	// Defaults to 0644 if not specified
	Mode string `yaml:"mode,omitempty" validate:"omitempty,file_mode"`

	// Files is a list of files to manage (new list-based syntax).
	// When populated, this takes precedence over the single-file fields above.
	// Each entry represents one file to manage.
	Files []FileSpec `yaml:"files,omitempty"`
}

// FileSpec represents a single file in the list-based ManagedFile structure.
type FileSpec struct {
	// Source is inline content for this file
	Source string `yaml:"source,omitempty"`

	// SourceFile is a path to an external file for this file
	SourceFile string `yaml:"sourceFile,omitempty"`

	// Destination is the absolute path for this file
	Destination string `yaml:"destination" validate:"required"`

	// Template indicates if the source should be processed as a Go template
	Template bool `yaml:"template,omitempty"`

	// Mode is the file permissions for this file
	Mode string `yaml:"mode,omitempty"`
}

// Validate implements Resource.Validate.
func (r ManagedFile) Validate() error {
	// Standard struct validation
	if err := ValidateStruct(r); err != nil {
		return err
	}

	// Custom validation for single-file syntax (backward compatibility)
	if len(r.Spec.Files) == 0 {
		// Single-file mode: exactly one of Source or SourceFile must be set
		hasSource := r.Spec.Source != ""
		hasSourceFile := r.Spec.SourceFile != ""

		if !hasSource && !hasSourceFile {
			return fmt.Errorf("ManagedFile.spec: exactly one of 'source' (inline) or 'sourceFile' (external file) is required")
		}

		if hasSource && hasSourceFile {
			return fmt.Errorf("ManagedFile.spec: 'source' and 'sourceFile' are mutually exclusive, use exactly one")
		}
	}

	return nil
}

// GetFiles returns the list of files to manage.
// If Files is populated (new syntax), returns that.
// Otherwise, converts the single-file fields to a list (backward compatibility).
func (r ManagedFile) GetFiles() []FileSpec {
	if len(r.Spec.Files) > 0 {
		return r.Spec.Files
	}

	// Convert single-file to list for backward compatibility
	return []FileSpec{{
		Source:      r.Spec.Source,
		SourceFile:  r.Spec.SourceFile,
		Destination: r.Spec.Destination,
		Template:   r.Spec.Template,
		Mode:       r.Spec.Mode,
	}}
}
