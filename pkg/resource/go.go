package resource

// GoPackages defines Go modules to install via `go install`.
type GoPackages struct {
	BaseResource `yaml:",inline"`
	Spec         GoPackagesSpec `yaml:"spec" validate:"required"`
}

// GoPackagesSpec contains the GoPackages configuration.
type GoPackagesSpec struct {
	// Packages to install
	Packages []GoPackage `yaml:"packages" validate:"required,dive"`
}

// GoPackage represents a Go module to install.
type GoPackage struct {
	// Module is the full module path (e.g., "golang.org/x/tools/gopls")
	Module string `yaml:"module" validate:"required"`

	// Version is the version to install (e.g., "latest", "v0.15.0")
	Version string `yaml:"version,omitempty"`
}

// Validate implements Resource.Validate.
func (r GoPackages) Validate() error {
	return ValidateStruct(r)
}

// ToGroup implements Resource.ToGroup.
func (r GoPackages) ToGroup() ResourceGroup {
	items := make([]ResourceItem, 0, len(r.Spec.Packages))

	for _, p := range r.Spec.Packages {
		items = append(items, ResourceItem{
			Name:    p.Module,
			Version: p.Version,
		})
	}

	return ResourceGroup{
		Kind:      r.Kind,
		Name:      r.Metadata.Name,
		Namespace: r.Metadata.GetNamespace(),
		Items:     items,
		RawSpec:   r.Spec,
	}
}
