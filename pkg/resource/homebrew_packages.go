package resource

import (
	"log/slog"
)

// HomeBrewPackages defines Homebrew formulae (command-line packages).
type HomeBrewPackages struct {
	BaseResource `yaml:",inline"`
	Spec         HomeBrewPackagesSpec `yaml:"spec" validate:"required"`
}

// HomeBrewPackagesSpec contains the HomeBrewPackages configuration.
type HomeBrewPackagesSpec struct {
	// Formulae to install (command-line tools)
	Formulae []Package `yaml:"formulae,omitempty" validate:"dive"`
}

// Validate implements Resource.Validate.
func (r HomeBrewPackages) Validate() error {
	return ValidateStruct(r)
}

// ToGroup implements Resource.ToGroup.
func (r HomeBrewPackages) ToGroup() ResourceGroup {
	items := make([]ResourceItem, 0)

	slog.Debug("HomeBrewPackages.ToGroup",
		"group", r.Metadata.Name,
		"formulae", len(r.Spec.Formulae),
	)

	// Add formulae as items
	for _, f := range r.Spec.Formulae {
		items = append(items, ResourceItem{
			Name:    f.Name,
			Version: f.Version,
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
