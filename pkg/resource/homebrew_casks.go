package resource

import (
	"log/slog"
)

// HomeBrewCasks defines Homebrew casks (GUI applications).
type HomeBrewCasks struct {
	BaseResource `yaml:",inline"`
	Spec         HomeBrewCasksSpec `yaml:"spec" validate:"required"`
}

// HomeBrewCasksSpec contains the HomeBrewCasks configuration.
type HomeBrewCasksSpec struct {
	Casks []Package `yaml:"casks,omitempty" validate:"dive"`
}

// Validate implements Resource.Validate.
func (r HomeBrewCasks) Validate() error {
	return ValidateStruct(r)
}

// ToGroup implements Resource.ToGroup.
func (r HomeBrewCasks) ToGroup() ResourceGroup {
	items := make([]ResourceItem, 0)

	slog.Debug("HomeBrewCasks.ToGroup",
		"group", r.Metadata.Name,
		"casks", len(r.Spec.Casks),
	)

	for _, c := range r.Spec.Casks {
		// Use plain cask name; provider will treat casks based on group.Kind
		items = append(items, ResourceItem{
			Name:    c.Name,
			Version: c.Version,
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
