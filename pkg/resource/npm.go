package resource

// NpmPackages defines global npm packages to install.
type NpmPackages struct {
	BaseResource `yaml:",inline"`
	Spec         NpmPackagesSpec `yaml:"spec" validate:"required"`
}

// NpmPackagesSpec contains the NpmPackages configuration.
type NpmPackagesSpec struct {
	// Packages to install globally
	Packages []Package `yaml:"packages" validate:"required,dive"`
}

// Validate implements Resource.Validate.
func (r NpmPackages) Validate() error {
	if err := ValidateStruct(r); err != nil {
		return err
	}
	return validateDependsOnAddresses(r.Metadata.DependsOn)
}

// ToGroup implements Resource.ToGroup.
func (r NpmPackages) ToGroup() ResourceGroup {
	items := make([]ResourceItem, 0, len(r.Spec.Packages))

	for _, p := range r.Spec.Packages {
		items = append(items, ResourceItem{
			Name:    p.Name,
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
