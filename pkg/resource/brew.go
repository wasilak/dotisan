package resource

// BrewPackages defines Homebrew packages to install.
// This includes formulae, casks, and taps.
type BrewPackages struct {
	BaseResource `yaml:",inline"`
	Spec         BrewPackagesSpec `yaml:"spec" validate:"required"`
}

// BrewPackagesSpec contains the BrewPackages configuration.
type BrewPackagesSpec struct {
	// Taps to add (optional)
	Taps []Tap `yaml:"taps,omitempty" validate:"dive"`

	// Formulae to install (command-line tools)
	Formulae []Package `yaml:"formulae,omitempty" validate:"dive"`

	// Casks to install (GUI applications)
	Casks []Package `yaml:"casks,omitempty" validate:"dive"`
}

// Tap represents a Homebrew tap to add.
type Tap struct {
	// Name is the tap name (e.g., "homebrew/cask-fonts")
	Name string `yaml:"name" validate:"required"`
}

// Package represents a package to install.
type Package struct {
	// Name is the package name
	Name string `yaml:"name" validate:"required"`

	// Version is optional; if specified, the provider may warn if unavailable
	Version string `yaml:"version,omitempty"`
}

// Validate implements Resource.Validate.
func (r BrewPackages) Validate() error {
	return ValidateStruct(r)
}
