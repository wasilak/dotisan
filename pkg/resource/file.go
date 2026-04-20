package resource

// ManagedFile defines a single file to manage.
// The file can be templated or static.
type ManagedFile struct {
	BaseResource `yaml:",inline"`
	Spec         ManagedFileSpec `yaml:"spec" validate:"required"`
}

// ManagedFileSpec contains the ManagedFile configuration.
type ManagedFileSpec struct {
	// Source is the path to the source file or template
	// Relative to the dotfiles root
	Source string `yaml:"source" validate:"required"`

	// Destination is the absolute path where the file should be placed
	// Can contain template expressions like "{{ .Values.user.home }}/.zshrc"
	Destination string `yaml:"destination" validate:"required"`

	// Template indicates if the source should be processed as a Go template
	Template bool `yaml:"template,omitempty"`

	// Mode is the file permissions (e.g., "0644", "0755")
	// Defaults to 0644 if not specified
	Mode string `yaml:"mode,omitempty" validate:"omitempty,file_mode"`
}

// Validate implements Resource.Validate.
func (r ManagedFile) Validate() error {
	return ValidateStruct(r)
}
