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
// Taps are not represented as items; they are considered group-level infra.
func (r HomeBrewTaps) ToGroup() ResourceGroup {
	// No items for taps — providers will read RawSpec to act on taps
	return ResourceGroup{
		Kind:    r.Kind,
		Name:    r.Metadata.Name,
		Items:   []ResourceItem{},
		RawSpec: r.Spec,
	}
}
