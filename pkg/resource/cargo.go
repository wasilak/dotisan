package resource

// CargoPackages defines Rust crates to install via `cargo install`.
type CargoPackages struct {
	BaseResource `yaml:",inline"`
	Spec         CargoPackagesSpec `yaml:"spec" validate:"required"`
}

// CargoPackagesSpec contains the CargoPackages configuration.
type CargoPackagesSpec struct {
	// Packages to install
	Packages []Package `yaml:"packages" validate:"required,dive"`
}

// Validate implements Resource.Validate.
func (r CargoPackages) Validate() error {
	return ValidateStruct(r)
}
