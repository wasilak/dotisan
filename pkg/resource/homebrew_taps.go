package resource

// HomeBrewTaps defines Homebrew taps to add.
type HomeBrewTaps struct {
	BaseResource `yaml:",inline"`
	Spec         HomeBrewTapsSpec `yaml:"spec" validate:"required"`
}

// HomeBrewTapsSpec contains the HomeBrewTaps configuration.
type HomeBrewTapsSpec struct {
	Taps []Tap `yaml:"taps,omitempty" validate:"dive"`
}

// Validate implements Resource.Validate.
func (r HomeBrewTaps) Validate() error {
	if err := ValidateStruct(r); err != nil {
		return err
	}
	return validateDependsOnAddresses(r.Metadata.DependsOn)
}

// ToGroup implements Resource.ToGroup.
func (r HomeBrewTaps) ToGroup() ResourceGroup[any] {
	items := make([]ResourceItem, 0, len(r.Spec.Taps))
	for _, t := range r.Spec.Taps {
		items = append(items, ResourceItem{Name: t.Name})
	}
	return ResourceGroup[any]{
		Kind:  r.Kind,
		Name:  r.Metadata.Name,
		Items: items,
	}
}
