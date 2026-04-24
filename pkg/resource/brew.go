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

// ToGroup implements Resource.ToGroup.
func (r BrewPackages) ToGroup() ResourceGroup {
	items := make([]ResourceItem, 0)

	// Add formulae as items
	for _, f := range r.Spec.Formulae {
		items = append(items, ResourceItem{
			Name:    f.Name,
			Version: f.Version,
		})
	}

	// Add casks as items
	for _, c := range r.Spec.Casks {
		items = append(items, ResourceItem{
			Name:    c.Name + " (cask)",
			Version: c.Version,
		})
	}

	// Note: Taps are not items - they're infrastructure for the group

	return ResourceGroup{
		Kind:      r.Kind,
		Name:      r.Metadata.Name,
		Namespace: r.Metadata.GetNamespace(),
		Items:     items,
		RawSpec:   r.Spec,
	}
}
