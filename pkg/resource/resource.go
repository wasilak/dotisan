// Package resource provides the core resource types and interfaces for dotisan.
//
// Resources follow a Kubernetes-style declarative model where each resource
// has apiVersion, kind, metadata, and a kind-specific spec.
//
// Example resource YAML:
//
//	apiVersion: dotisan/v1
//	kind: BrewPackages
//	metadata:
//	  name: core-tools
//	  namespace: default
//	spec:
//	  formulae:
//	    - name: ripgrep
//	    - name: fd
package resource

import (
	"fmt"
	"regexp"

	"github.com/go-playground/validator/v10"
)

// init registers custom validators.
func init() {
	// Register file_mode validator (e.g., "0644", "0755")
	validate.RegisterValidation("file_mode", validateFileMode)
}

// fileModeRegex matches valid Unix file modes (4-digit octal like 0644, 0755)
var fileModeRegex = regexp.MustCompile(`^[0-7]{4}$`)

// validateFileMode validates a file mode string.
func validateFileMode(fl validator.FieldLevel) bool {
	mode := fl.Field().String()
	if mode == "" {
		return true // Empty is allowed (will use default)
	}
	return fileModeRegex.MatchString(mode)
}

// Resource is the interface implemented by all resource types.
// It provides common access to resource metadata and validation.
type Resource interface {
	// GetAPIVersion returns the API version (e.g., "github.com/wasilak/dotisan/v1")
	GetAPIVersion() string

	// GetKind returns the resource kind (e.g., "HomeBrewPackages")
	GetKind() string

	// GetMetadata returns the resource metadata
	GetMetadata() Metadata

	// Validate validates the resource spec and returns any validation errors
	Validate() error

	// ToGroup converts this resource to a ResourceGroup representation
	// This extracts items from the spec and creates the 3-level hierarchy
	ToGroup() ResourceGroup
}

// Metadata contains common metadata for all resources.
type Metadata struct {
	// Name is the unique name for this resource within its namespace
	Name string `yaml:"name" validate:"required,min=1,max=253"`

	// Namespace is a logical grouping (defaults to "default")
	Namespace string `yaml:"namespace,omitempty" validate:"omitempty,min=1,max=253"`

	// Labels are optional key-value pairs for resource organization
	Labels map[string]string `yaml:"labels,omitempty"`

	// Annotations are optional metadata for tooling
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

// GetNamespace returns the namespace or "default" if not set.
func (m Metadata) GetNamespace() string {
	if m.Namespace == "" {
		return "default"
	}
	return m.Namespace
}

// ResourceID returns a unique identifier for the resource (namespace/name).
func (m Metadata) ResourceID() string {
	return fmt.Sprintf("%s/%s", m.GetNamespace(), m.Name)
}

// BaseResource provides common fields embedded in all resource structs.
// It partially implements the Resource interface.
type BaseResource struct {
	APIVersion string   `yaml:"apiVersion" validate:"required"`
	Kind       string   `yaml:"kind" validate:"required"`
	Metadata   Metadata `yaml:"metadata" validate:"required"`
}

// GetAPIVersion implements Resource.GetAPIVersion.
func (r BaseResource) GetAPIVersion() string {
	return r.APIVersion
}

// GetKind implements Resource.GetKind.
func (r BaseResource) GetKind() string {
	return r.Kind
}

// GetMetadata implements Resource.GetMetadata.
func (r BaseResource) GetMetadata() Metadata {
	return r.Metadata
}

// validate is a shared validator instance.
var validate = validator.New()

// ValidateStruct validates a struct using go-playground/validator.
// This is a helper for resource implementations to use in their Validate() method.
func ValidateStruct(s interface{}) error {
	if err := validate.Struct(s); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			return fmt.Errorf("validation failed: %v", validationErrors)
		}
		return err
	}
	return nil
}

// SupportedAPIVersion is the current supported API version.
const SupportedAPIVersion = "github.com/wasilak/dotisan/v1"

// IsSupportedAPIVersion checks if the given API version is supported.
func IsSupportedAPIVersion(version string) bool {
	return version == SupportedAPIVersion
}
