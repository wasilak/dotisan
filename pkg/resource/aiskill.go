package resource

// AISkillPackages defines AI skill packages to install via the `skills` CLI.
type AISkillPackages struct {
	BaseResource `yaml:",inline"`
	Spec         AISkillPackagesSpec `yaml:"spec" validate:"required"`
}

// AISkillPackagesSpec contains the AISkillPackages configuration.
type AISkillPackagesSpec struct {
	// Packages to install
	Packages []AISkillPackage `yaml:"packages" validate:"required,dive"`
}

// AISkillPackage represents a skill package from a GitHub repository.
type AISkillPackage struct {
	// Source is the GitHub repo slug or full URL (e.g. "Ar9av/obsidian-wiki")
	Source string `yaml:"source" validate:"required"`

	// Targets lists which agents to install for (e.g. ["claude", "opencode"]).
	// If empty or omitted, installs for all detected agents.
	Targets []string `yaml:"targets,omitempty"`
}

// Validate implements Resource.Validate.
func (r AISkillPackages) Validate() error {
	if err := ValidateStruct(r); err != nil {
		return err
	}
	return validateDependsOnAddresses(r.Metadata.DependsOn)
}

// ToGroup implements Resource.ToGroup.
func (r AISkillPackages) ToGroup() ResourceGroup[any] {
	items := make([]ResourceItem, 0, len(r.Spec.Packages))

	for _, p := range r.Spec.Packages {
		items = append(items, ResourceItem{
			Name: p.Source,
		})
	}

	return ResourceGroup[any]{
		Kind:    r.Kind,
		Name:    r.Metadata.Name,
		Items:   items,
		RawSpec: r.Spec,
	}
}
