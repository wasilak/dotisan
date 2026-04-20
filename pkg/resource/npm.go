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
	return ValidateStruct(r)
}
