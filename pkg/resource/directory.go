package resource

// ManagedDirectory defines a directory to manage recursively.
// Can optionally clean (remove files not in source).
type ManagedDirectory struct {
	BaseResource `yaml:",inline"`
	Spec         ManagedDirectorySpec `yaml:"spec" validate:"required"`
}

// ManagedDirectorySpec contains the ManagedDirectory configuration.
type ManagedDirectorySpec struct {
	// Source is the path to the source directory
	// Relative to the dotfiles root
	Source string `yaml:"source" validate:"required"`

	// Destination is the absolute path where the directory should be placed
	// Can contain template expressions
	Destination string `yaml:"destination" validate:"required"`

	// Recursive indicates if subdirectories should be synced
	Recursive bool `yaml:"recursive,omitempty"`

	// Clean indicates if files at destination not in source should be removed
	Clean bool `yaml:"clean,omitempty"`

	// Exclude is a list of glob patterns to exclude from sync
	Exclude []string `yaml:"exclude,omitempty"`
}

// Validate implements Resource.Validate.
func (r ManagedDirectory) Validate() error {
	return ValidateStruct(r)
}
